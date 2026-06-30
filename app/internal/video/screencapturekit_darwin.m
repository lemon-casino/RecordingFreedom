#import "screencapturekit_darwin.h"

#import <AudioToolbox/AudioToolbox.h>
#import <AVFoundation/AVFoundation.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <Foundation/Foundation.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <dispatch/dispatch.h>

#include <stdlib.h>
#include <string.h>
#include <math.h>

struct RFSCKSession {
	void *impl;
};

static char *rf_sck_copy_message(NSString *message) {
	if (message == nil || message.length == 0) {
		return NULL;
	}
	const char *utf8 = [message UTF8String];
	if (utf8 == NULL) {
		return NULL;
	}
	return strdup(utf8);
}

static void rf_sck_set_error(char **out, NSString *message) {
	if (out == NULL) {
		return;
	}
	*out = rf_sck_copy_message(message);
}

static NSString *rf_sck_error_message(NSError *error, NSString *fallback) {
	if (error == nil) {
		return fallback;
	}
	if (error.localizedDescription.length > 0) {
		return error.localizedDescription;
	}
	return fallback;
}

static int64_t rf_sck_elapsed_ms(CMTime start, CMTime value) {
	if (!CMTIME_IS_NUMERIC(start) || !CMTIME_IS_NUMERIC(value)) {
		return 0;
	}
	CMTime elapsed = CMTimeSubtract(value, start);
	Float64 seconds = CMTimeGetSeconds(elapsed);
	if (!isfinite(seconds) || seconds < 0) {
		return 0;
	}
	return (int64_t)(seconds * 1000.0);
}

@interface RFScreenCaptureSession : NSObject <SCStreamDelegate, SCStreamOutput>
@property(nonatomic, assign) int targetKind;
@property(nonatomic, assign) uint32_t targetID;
@property(nonatomic, copy) NSString *outputPath;
@property(nonatomic, copy) NSString *quality;
@property(nonatomic, assign) int requestedFPS;
@property(nonatomic, assign) BOOL captureCursor;
@property(nonatomic, assign) BOOL captureSystemAudio;
@property(nonatomic, strong) SCStream *stream;
@property(nonatomic, strong) AVAssetWriter *writer;
@property(nonatomic, strong) AVAssetWriterInput *videoInput;
@property(nonatomic, strong) AVAssetWriterInput *audioInput;
@property(nonatomic, strong) dispatch_queue_t sampleQueue;
@property(nonatomic, assign) BOOL started;
@property(nonatomic, assign) BOOL stopped;
@property(nonatomic, assign) BOOL paused;
@property(nonatomic, assign) BOOL writerStarted;
@property(nonatomic, assign) CMTime firstPTS;
@property(nonatomic, assign) CMTime lastPTS;
@property(nonatomic, assign) CMTime firstAudioPTS;
@property(nonatomic, assign) CMTime lastAudioPTS;
@property(nonatomic, assign) int width;
@property(nonatomic, assign) int height;
@property(nonatomic, assign) int frameRate;
@property(nonatomic, assign) int64_t framesWritten;
@property(nonatomic, assign) int64_t droppedFrames;
@property(nonatomic, assign) int64_t appendFailures;
@property(nonatomic, assign) int audioSampleRate;
@property(nonatomic, assign) int64_t audioSamplesWritten;
@property(nonatomic, assign) int64_t audioDroppedSamples;
@property(nonatomic, assign) int64_t audioAppendFailures;
@property(nonatomic, copy) NSString *lastMessage;
@property(nonatomic, copy) NSString *lastAudioMessage;
@end

@implementation RFScreenCaptureSession

- (instancetype)initWithTargetKind:(int)targetKind
						  targetID:(uint32_t)targetID
					   outputPath:(NSString *)outputPath
							  fps:(int)fps
					captureCursor:(BOOL)captureCursor
				captureSystemAudio:(BOOL)captureSystemAudio
						  quality:(NSString *)quality {
	self = [super init];
	if (self == nil) {
		return nil;
	}
	_targetKind = targetKind;
	_targetID = targetID;
	_outputPath = [outputPath copy];
	_requestedFPS = fps > 0 ? fps : 30;
	_captureCursor = captureCursor;
	_captureSystemAudio = captureSystemAudio;
	_quality = quality.length > 0 ? [quality copy] : @"balanced";
	_sampleQueue = dispatch_queue_create("casino.lemon.recordingfreedom.sck-video", DISPATCH_QUEUE_SERIAL);
	_firstPTS = kCMTimeInvalid;
	_lastPTS = kCMTimeInvalid;
	_firstAudioPTS = kCMTimeInvalid;
	_lastAudioPTS = kCMTimeInvalid;
	_frameRate = _requestedFPS;
	return self;
}

- (BOOL)startWithError:(NSString **)errorMessage {
	if (self.started) {
		if (errorMessage != NULL) {
			*errorMessage = @"ScreenCaptureKit session is already started";
		}
		return NO;
	}
	if (self.stopped) {
		if (errorMessage != NULL) {
			*errorMessage = @"ScreenCaptureKit session is stopped";
		}
		return NO;
	}
	if (self.outputPath.length == 0) {
		if (errorMessage != NULL) {
			*errorMessage = @"ScreenCaptureKit output path is required";
		}
		return NO;
	}
	if (@available(macOS 12.3, *)) {
		return [self startAvailableWithError:errorMessage];
	}
	if (errorMessage != NULL) {
		*errorMessage = @"ScreenCaptureKit requires macOS 12.3 or newer";
	}
	return NO;
}

- (BOOL)startAvailableWithError:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	NSError *prepareError = nil;
	if (![self prepareOutputPath:&prepareError]) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(prepareError, @"Could not prepare ScreenCaptureKit output path");
		}
		return NO;
	}

	SCContentFilter *filter = [self contentFilterWithError:errorMessage];
	if (filter == nil) {
		return NO;
	}

	if (self.width <= 0 || self.height <= 0) {
		if (errorMessage != NULL) {
			*errorMessage = [NSString stringWithFormat:@"ScreenCaptureKit target %u has invalid dimensions", self.targetID];
		}
		return NO;
	}

	NSError *writerError = nil;
	if (![self configureWriterWithWidth:self.width height:self.height error:&writerError]) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(writerError, @"Could not create AVAssetWriter");
		}
		return NO;
	}

	SCStreamConfiguration *configuration = [[SCStreamConfiguration alloc] init];
	configuration.width = self.width;
	configuration.height = self.height;
	configuration.minimumFrameInterval = CMTimeMake(1, self.requestedFPS);
	configuration.queueDepth = 6;
	configuration.pixelFormat = kCVPixelFormatType_32BGRA;
	configuration.showsCursor = self.captureCursor;
	if (self.captureSystemAudio) {
		if (@available(macOS 13.0, *)) {
			configuration.capturesAudio = YES;
		} else {
			if (errorMessage != NULL) {
				*errorMessage = @"ScreenCaptureKit system audio capture requires macOS 13.0 or newer";
			}
			return NO;
		}
	}

	self.stream = [[SCStream alloc] initWithFilter:filter configuration:configuration delegate:self];

	NSError *outputError = nil;
	if (![self.stream addStreamOutput:self type:SCStreamOutputTypeScreen sampleHandlerQueue:self.sampleQueue error:&outputError]) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(outputError, @"Could not attach ScreenCaptureKit stream output");
		}
		return NO;
	}
	if (self.captureSystemAudio) {
		if (@available(macOS 13.0, *)) {
			if (![self.stream addStreamOutput:self type:SCStreamOutputTypeAudio sampleHandlerQueue:self.sampleQueue error:&outputError]) {
				if (errorMessage != NULL) {
					*errorMessage = rf_sck_error_message(outputError, @"Could not attach ScreenCaptureKit audio output");
				}
				return NO;
			}
		}
	}

	__block NSError *startError = nil;
	dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
	[self.stream startCaptureWithCompletionHandler:^(NSError * _Nullable error) {
		startError = error;
		dispatch_semaphore_signal(semaphore);
	}];
	if (dispatch_semaphore_wait(semaphore, dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_SEC)) != 0) {
		if (errorMessage != NULL) {
			*errorMessage = @"Timed out starting ScreenCaptureKit stream";
		}
		return NO;
	}
	if (startError != nil) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(startError, @"ScreenCaptureKit start failed");
		}
		return NO;
	}

	self.started = YES;
	self.lastMessage = @"ScreenCaptureKit stream started";
	return YES;
}

- (SCContentFilter *)contentFilterWithError:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	__block SCShareableContent *content = nil;
	__block NSError *contentError = nil;
	dispatch_semaphore_t semaphore = dispatch_semaphore_create(0);
	[SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent * _Nullable shareableContent, NSError * _Nullable error) {
		content = shareableContent;
		contentError = error;
		dispatch_semaphore_signal(semaphore);
	}];
	if (dispatch_semaphore_wait(semaphore, dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_SEC)) != 0) {
		if (errorMessage != NULL) {
			*errorMessage = @"Timed out resolving ScreenCaptureKit display";
		}
		return nil;
	}
	if (contentError != nil) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(contentError, @"ScreenCaptureKit shareable content failed");
		}
		return nil;
	}
	if (self.targetKind == RF_SCK_TARGET_DISPLAY) {
		return [self displayFilterWithContent:content error:errorMessage];
	}
	if (self.targetKind == RF_SCK_TARGET_WINDOW) {
		return [self windowFilterWithContent:content error:errorMessage];
	}
	if (self.targetKind == RF_SCK_TARGET_APPLICATION) {
		return [self applicationFilterWithContent:content error:errorMessage];
	}
	if (errorMessage != NULL) {
		*errorMessage = [NSString stringWithFormat:@"ScreenCaptureKit target kind %d is not supported", self.targetKind];
	}
	return nil;
}

- (SCContentFilter *)displayFilterWithContent:(SCShareableContent *)content error:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	for (SCDisplay *display in content.displays) {
		if ((uint32_t)display.displayID == self.targetID) {
			self.width = (int)display.width;
			self.height = (int)display.height;
			return [[SCContentFilter alloc] initWithDisplay:display
									  excludingApplications:@[]
										exceptingWindows:@[]];
		}
	}
	if (errorMessage != NULL) {
		*errorMessage = [NSString stringWithFormat:@"ScreenCaptureKit display %u was not found", self.targetID];
	}
	return nil;
}

- (SCContentFilter *)filterWithWindow:(SCWindow *)window error:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	if (window == nil) {
		if (errorMessage != NULL) {
			*errorMessage = @"ScreenCaptureKit window target is nil";
		}
		return nil;
	}
	CGRect frame = window.frame;
	self.width = (int)ceil(frame.size.width);
	self.height = (int)ceil(frame.size.height);
	return [[SCContentFilter alloc] initWithDesktopIndependentWindow:window];
}

- (SCContentFilter *)windowFilterWithContent:(SCShareableContent *)content error:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	for (SCWindow *window in content.windows) {
		if ((uint32_t)window.windowID == self.targetID) {
			return [self filterWithWindow:window error:errorMessage];
		}
	}
	if (errorMessage != NULL) {
		*errorMessage = [NSString stringWithFormat:@"ScreenCaptureKit window %u was not found", self.targetID];
	}
	return nil;
}

- (SCContentFilter *)applicationFilterWithContent:(SCShareableContent *)content error:(NSString **)errorMessage API_AVAILABLE(macos(12.3)) {
	SCWindow *bestWindow = nil;
	double bestArea = 0.0;
	for (SCWindow *window in content.windows) {
		SCRunningApplication *application = window.owningApplication;
		if (application == nil || (uint32_t)application.processID != self.targetID) {
			continue;
		}
		CGRect frame = window.frame;
		double area = frame.size.width * frame.size.height;
		if (area <= 0.0) {
			continue;
		}
		if (bestWindow == nil || area > bestArea) {
			bestWindow = window;
			bestArea = area;
		}
	}
	if (bestWindow != nil) {
		return [self filterWithWindow:bestWindow error:errorMessage];
	}
	if (errorMessage != NULL) {
		*errorMessage = [NSString stringWithFormat:@"ScreenCaptureKit found no visible window for application pid %u", self.targetID];
	}
	return nil;
}

- (BOOL)prepareOutputPath:(NSError **)error {
	NSString *directory = [self.outputPath stringByDeletingLastPathComponent];
	if (directory.length > 0) {
		if (![[NSFileManager defaultManager] createDirectoryAtPath:directory withIntermediateDirectories:YES attributes:nil error:error]) {
			return NO;
		}
	}
	if ([[NSFileManager defaultManager] fileExistsAtPath:self.outputPath]) {
		return [[NSFileManager defaultManager] removeItemAtPath:self.outputPath error:error];
	}
	return YES;
}

- (BOOL)configureWriterWithWidth:(int)width height:(int)height error:(NSError **)error {
	NSURL *url = [NSURL fileURLWithPath:self.outputPath];
	self.writer = [[AVAssetWriter alloc] initWithURL:url fileType:AVFileTypeMPEG4 error:error];
	if (self.writer == nil) {
		return NO;
	}

	NSNumber *bitrate = @([self bitrateForWidth:width height:height]);
	NSDictionary *compression = @{
		AVVideoAverageBitRateKey: bitrate,
		AVVideoExpectedSourceFrameRateKey: @(self.requestedFPS),
		AVVideoProfileLevelKey: AVVideoProfileLevelH264HighAutoLevel,
	};
	NSDictionary *settings = @{
		AVVideoCodecKey: AVVideoCodecTypeH264,
		AVVideoWidthKey: @(width),
		AVVideoHeightKey: @(height),
		AVVideoCompressionPropertiesKey: compression,
	};
	self.videoInput = [AVAssetWriterInput assetWriterInputWithMediaType:AVMediaTypeVideo outputSettings:settings];
	self.videoInput.expectsMediaDataInRealTime = YES;
	if (![self.writer canAddInput:self.videoInput]) {
		if (error != NULL) {
			*error = [NSError errorWithDomain:@"RecordingFreedom.ScreenCaptureKit"
										 code:1001
									 userInfo:@{NSLocalizedDescriptionKey: @"AVAssetWriter cannot add H.264 video input"}];
		}
		return NO;
	}
	[self.writer addInput:self.videoInput];
	if (self.captureSystemAudio) {
		NSDictionary *audioSettings = @{
			AVFormatIDKey: @(kAudioFormatMPEG4AAC),
			AVSampleRateKey: @48000,
			AVNumberOfChannelsKey: @2,
			AVEncoderBitRateKey: @128000,
		};
		self.audioInput = [AVAssetWriterInput assetWriterInputWithMediaType:AVMediaTypeAudio outputSettings:audioSettings];
		self.audioInput.expectsMediaDataInRealTime = YES;
		if (![self.writer canAddInput:self.audioInput]) {
			if (error != NULL) {
				*error = [NSError errorWithDomain:@"RecordingFreedom.ScreenCaptureKit"
											 code:1003
										 userInfo:@{NSLocalizedDescriptionKey: @"AVAssetWriter cannot add AAC audio input"}];
			}
			return NO;
		}
		[self.writer addInput:self.audioInput];
		self.audioSampleRate = 48000;
	}
	return YES;
}

- (int)bitrateForWidth:(int)width height:(int)height {
	double bitsPerPixel = 0.08;
	if ([self.quality isEqualToString:@"standard"]) {
		bitsPerPixel = 0.05;
	} else if ([self.quality isEqualToString:@"high"]) {
		bitsPerPixel = 0.12;
	}
	double estimated = (double)width * (double)height * (double)self.requestedFPS * bitsPerPixel;
	if (estimated < 2000000.0) {
		estimated = 2000000.0;
	}
	if (estimated > 45000000.0) {
		estimated = 45000000.0;
	}
	return (int)estimated;
}

- (BOOL)pauseWithError:(NSString **)errorMessage {
	if (!self.started || self.stopped) {
		return YES;
	}
	self.paused = YES;
	self.lastMessage = @"ScreenCaptureKit video paused";
	return YES;
}

- (BOOL)resumeWithError:(NSString **)errorMessage {
	if (!self.started || self.stopped) {
		return YES;
	}
	self.paused = NO;
	self.lastMessage = @"ScreenCaptureKit video resumed";
	return YES;
}

- (BOOL)stopWithError:(NSString **)errorMessage {
	if (self.stopped) {
		return YES;
	}
	self.stopped = YES;

	__block NSError *stopError = nil;
	if (self.stream != nil) {
		dispatch_semaphore_t stopSemaphore = dispatch_semaphore_create(0);
		[self.stream stopCaptureWithCompletionHandler:^(NSError * _Nullable error) {
			stopError = error;
			dispatch_semaphore_signal(stopSemaphore);
		}];
		if (dispatch_semaphore_wait(stopSemaphore, dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_SEC)) != 0) {
			if (errorMessage != NULL) {
				*errorMessage = @"Timed out stopping ScreenCaptureKit stream";
			}
			return NO;
		}
	}

	__block NSError *finishError = nil;
	dispatch_sync(self.sampleQueue, ^{
		finishError = [self finishWriter];
	});

	if (stopError != nil) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(stopError, @"ScreenCaptureKit stop failed");
		}
		return NO;
	}
	if (finishError != nil) {
		if (errorMessage != NULL) {
			*errorMessage = rf_sck_error_message(finishError, @"AVAssetWriter finish failed");
		}
		return NO;
	}

	if (self.framesWritten == 0 && self.lastMessage.length == 0) {
		self.lastMessage = @"ScreenCaptureKit stopped before a video frame was written";
	} else if (self.lastMessage.length == 0) {
		self.lastMessage = @"ScreenCaptureKit stream stopped";
	}
	return YES;
}

- (NSError *)finishWriter {
	if (!self.writerStarted) {
		return nil;
	}
	if (self.writer.status == AVAssetWriterStatusCompleted) {
		return nil;
	}
	if (self.writer.status == AVAssetWriterStatusFailed || self.writer.status == AVAssetWriterStatusCancelled) {
		return self.writer.error;
	}
	[self.videoInput markAsFinished];
	if (self.audioInput != nil) {
		[self.audioInput markAsFinished];
	}
	dispatch_semaphore_t writerSemaphore = dispatch_semaphore_create(0);
	[self.writer finishWritingWithCompletionHandler:^{
		dispatch_semaphore_signal(writerSemaphore);
	}];
	if (dispatch_semaphore_wait(writerSemaphore, dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_SEC)) != 0) {
		return [NSError errorWithDomain:@"RecordingFreedom.ScreenCaptureKit"
								   code:1002
							   userInfo:@{NSLocalizedDescriptionKey: @"Timed out finalizing AVAssetWriter"}];
	}
	if (self.writer.status == AVAssetWriterStatusFailed || self.writer.status == AVAssetWriterStatusCancelled) {
		return self.writer.error;
	}
	return nil;
}

- (void)stream:(SCStream *)stream didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer ofType:(SCStreamOutputType)type {
	if (@available(macOS 13.0, *)) {
		if (type == SCStreamOutputTypeAudio) {
			[self handleAudioSampleBuffer:sampleBuffer];
			return;
		}
	}
	if (type != SCStreamOutputTypeScreen) {
		return;
	}
	if (self.paused || self.stopped) {
		self.droppedFrames++;
		return;
	}
	if (sampleBuffer == NULL || !CMSampleBufferIsValid(sampleBuffer) || !CMSampleBufferDataIsReady(sampleBuffer)) {
		self.droppedFrames++;
		return;
	}

	CMTime pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer);
	if (!CMTIME_IS_NUMERIC(pts)) {
		self.droppedFrames++;
		return;
	}

	if (!self.writerStarted) {
		if (![self.writer startWriting]) {
			self.appendFailures++;
			self.lastMessage = rf_sck_error_message(self.writer.error, @"AVAssetWriter could not start writing");
			return;
		}
		[self.writer startSessionAtSourceTime:pts];
		self.firstPTS = pts;
		self.writerStarted = YES;
	}

	if (!self.videoInput.readyForMoreMediaData) {
		self.droppedFrames++;
		return;
	}
	if (![self.videoInput appendSampleBuffer:sampleBuffer]) {
		self.appendFailures++;
		self.lastMessage = rf_sck_error_message(self.writer.error, @"AVAssetWriter append failed");
		return;
	}

	self.lastPTS = pts;
	self.framesWritten++;
}

- (void)handleAudioSampleBuffer:(CMSampleBufferRef)sampleBuffer API_AVAILABLE(macos(13.0)) {
	CMItemCount sampleCount = sampleBuffer == NULL ? 0 : CMSampleBufferGetNumSamples(sampleBuffer);
	if (sampleCount <= 0) {
		sampleCount = 1;
	}
	if (!self.captureSystemAudio || self.audioInput == nil) {
		self.audioDroppedSamples += sampleCount;
		return;
	}
	if (self.paused || self.stopped) {
		self.audioDroppedSamples += sampleCount;
		return;
	}
	if (sampleBuffer == NULL || !CMSampleBufferIsValid(sampleBuffer) || !CMSampleBufferDataIsReady(sampleBuffer)) {
		self.audioDroppedSamples += sampleCount;
		return;
	}

	CMTime pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer);
	if (!CMTIME_IS_NUMERIC(pts)) {
		self.audioDroppedSamples += sampleCount;
		return;
	}
	if (!self.writerStarted) {
		self.audioDroppedSamples += sampleCount;
		self.lastAudioMessage = @"ScreenCaptureKit audio arrived before the first video frame";
		return;
	}
	if (!self.audioInput.readyForMoreMediaData) {
		self.audioDroppedSamples += sampleCount;
		return;
	}
	if (![self.audioInput appendSampleBuffer:sampleBuffer]) {
		self.audioAppendFailures++;
		self.lastAudioMessage = rf_sck_error_message(self.writer.error, @"AVAssetWriter audio append failed");
		return;
	}

	if (!CMTIME_IS_NUMERIC(self.firstAudioPTS)) {
		self.firstAudioPTS = pts;
	}
	self.lastAudioPTS = pts;
	self.audioSamplesWritten += sampleCount;
}

- (void)stream:(SCStream *)stream didStopWithError:(NSError *)error {
	if (error != nil) {
		self.lastMessage = rf_sck_error_message(error, @"ScreenCaptureKit stream stopped with an error");
	}
}

- (void)copyDiagnostics:(RFSCKDiagnostics *)diagnostics {
	if (diagnostics == NULL) {
		return;
	}
	memset(diagnostics, 0, sizeof(RFSCKDiagnostics));
	diagnostics->enabled = 1;
	diagnostics->width = self.width;
	diagnostics->height = self.height;
	diagnostics->frameRate = self.frameRate;
	diagnostics->framesWritten = self.framesWritten;
	diagnostics->droppedFrames = self.droppedFrames;
	diagnostics->appendFailures = self.appendFailures;
	diagnostics->startOffsetMs = 0;
	diagnostics->endOffsetMs = rf_sck_elapsed_ms(self.firstPTS, self.lastPTS);
	diagnostics->durationMs = diagnostics->endOffsetMs;
	diagnostics->message = rf_sck_copy_message(self.lastMessage);
	if (self.captureSystemAudio) {
		diagnostics->audioEnabled = 1;
		diagnostics->audioSampleRate = self.audioSampleRate;
		diagnostics->audioSamplesWritten = self.audioSamplesWritten;
		diagnostics->audioDroppedSamples = self.audioDroppedSamples;
		diagnostics->audioAppendFailures = self.audioAppendFailures;
		diagnostics->audioStartOffsetMs = rf_sck_elapsed_ms(self.firstPTS, self.firstAudioPTS);
		diagnostics->audioEndOffsetMs = rf_sck_elapsed_ms(self.firstPTS, self.lastAudioPTS);
		diagnostics->audioDurationMs = rf_sck_elapsed_ms(self.firstAudioPTS, self.lastAudioPTS);
		if (self.lastAudioMessage.length == 0 && self.audioSamplesWritten > 0) {
			diagnostics->audioMessage = rf_sck_copy_message(@"ScreenCaptureKit system audio muxed into screen.mp4");
		} else {
			diagnostics->audioMessage = rf_sck_copy_message(self.lastAudioMessage);
		}
	}
}

@end

RFSCKSession *rf_sck_session_create(
	int target_kind,
	uint32_t target_id,
	const char *output_path,
	int fps,
	int capture_cursor,
	int capture_system_audio,
	const char *quality,
	char **error_message
) {
	if (target_kind != RF_SCK_TARGET_DISPLAY && target_kind != RF_SCK_TARGET_WINDOW && target_kind != RF_SCK_TARGET_APPLICATION) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit target kind is not supported");
		return NULL;
	}
	if (target_id == 0) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit target id is required");
		return NULL;
	}
	if (output_path == NULL || strlen(output_path) == 0) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit output path is required");
		return NULL;
	}
	NSString *outputPath = [NSString stringWithUTF8String:output_path];
	NSString *qualityValue = quality == NULL ? @"balanced" : [NSString stringWithUTF8String:quality];
	RFScreenCaptureSession *impl = [[RFScreenCaptureSession alloc] initWithTargetKind:target_kind
																			targetID:target_id
																		  outputPath:outputPath
																				 fps:fps
																	   captureCursor:capture_cursor != 0
																 captureSystemAudio:capture_system_audio != 0
																			 quality:qualityValue];
	if (impl == nil) {
		rf_sck_set_error(error_message, @"Could not allocate ScreenCaptureKit session");
		return NULL;
	}
	RFSCKSession *session = (RFSCKSession *)calloc(1, sizeof(RFSCKSession));
	if (session == NULL) {
		rf_sck_set_error(error_message, @"Could not allocate ScreenCaptureKit session reference");
		return NULL;
	}
	session->impl = (__bridge_retained void *)impl;
	return session;
}

int rf_sck_session_start(RFSCKSession *session, char **error_message) {
	if (session == NULL || session->impl == NULL) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit session is nil");
		return 0;
	}
	RFScreenCaptureSession *impl = (__bridge RFScreenCaptureSession *)session->impl;
	NSString *message = nil;
	BOOL ok = [impl startWithError:&message];
	if (!ok) {
		rf_sck_set_error(error_message, message);
	}
	return ok ? 1 : 0;
}

int rf_sck_session_pause(RFSCKSession *session, char **error_message) {
	if (session == NULL || session->impl == NULL) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit session is nil");
		return 0;
	}
	RFScreenCaptureSession *impl = (__bridge RFScreenCaptureSession *)session->impl;
	NSString *message = nil;
	BOOL ok = [impl pauseWithError:&message];
	if (!ok) {
		rf_sck_set_error(error_message, message);
	}
	return ok ? 1 : 0;
}

int rf_sck_session_resume(RFSCKSession *session, char **error_message) {
	if (session == NULL || session->impl == NULL) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit session is nil");
		return 0;
	}
	RFScreenCaptureSession *impl = (__bridge RFScreenCaptureSession *)session->impl;
	NSString *message = nil;
	BOOL ok = [impl resumeWithError:&message];
	if (!ok) {
		rf_sck_set_error(error_message, message);
	}
	return ok ? 1 : 0;
}

int rf_sck_session_stop(RFSCKSession *session, char **error_message) {
	if (session == NULL || session->impl == NULL) {
		rf_sck_set_error(error_message, @"ScreenCaptureKit session is nil");
		return 0;
	}
	RFScreenCaptureSession *impl = (__bridge RFScreenCaptureSession *)session->impl;
	NSString *message = nil;
	BOOL ok = [impl stopWithError:&message];
	if (!ok) {
		rf_sck_set_error(error_message, message);
	}
	return ok ? 1 : 0;
}

void rf_sck_session_diagnostics(RFSCKSession *session, RFSCKDiagnostics *diagnostics) {
	if (diagnostics == NULL) {
		return;
	}
	memset(diagnostics, 0, sizeof(RFSCKDiagnostics));
	if (session == NULL || session->impl == NULL) {
		return;
	}
	RFScreenCaptureSession *impl = (__bridge RFScreenCaptureSession *)session->impl;
	[impl copyDiagnostics:diagnostics];
}

void rf_sck_session_release(RFSCKSession *session) {
	if (session == NULL) {
		return;
	}
	if (session->impl != NULL) {
		RFScreenCaptureSession *impl = (__bridge_transfer RFScreenCaptureSession *)session->impl;
		impl = nil;
		session->impl = NULL;
	}
	free(session);
}

void rf_sck_free_string(char *value) {
	free(value);
}
