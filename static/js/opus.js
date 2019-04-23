///<reference path="d.ts/asm.d.ts" />
///<reference path="d.ts/libopus.d.ts" />
var OpusApplication;
(function (OpusApplication) {
    OpusApplication[OpusApplication["VoIP"] = 2048] = "VoIP";
    OpusApplication[OpusApplication["Audio"] = 2049] = "Audio";
    OpusApplication[OpusApplication["RestrictedLowDelay"] = 2051] = "RestrictedLowDelay";
})(OpusApplication || (OpusApplication = {}));
var OpusError;
(function (OpusError) {
    OpusError[OpusError["OK"] = 0] = "OK";
    OpusError[OpusError["BadArgument"] = -1] = "BadArgument";
    OpusError[OpusError["BufferTooSmall"] = -2] = "BufferTooSmall";
    OpusError[OpusError["InternalError"] = -3] = "InternalError";
    OpusError[OpusError["InvalidPacket"] = -4] = "InvalidPacket";
    OpusError[OpusError["Unimplemented"] = -5] = "Unimplemented";
    OpusError[OpusError["InvalidState"] = -6] = "InvalidState";
    OpusError[OpusError["AllocFail"] = -7] = "AllocFail";
})(OpusError || (OpusError = {}));
var Opus = (function () {
    function Opus() {
    }
    Opus.getVersion = function () {
        var ptr = _opus_get_version_string();
        return Pointer_stringify(ptr);
    };
    Opus.getMaxFrameSize = function (numberOfStreams) {
        if (numberOfStreams === void 0) { numberOfStreams = 1; }
        return (1275 * 3 + 7) * numberOfStreams;
    };
    Opus.getMinFrameDuration = function () {
        return 2.5;
    };
    Opus.getMaxFrameDuration = function () {
        return 60;
    };
    Opus.validFrameDuration = function (x) {
        return [2.5, 5, 10, 20, 40, 60].some(function (element) {
            return element == x;
        });
    };
    Opus.getMaxSamplesPerChannel = function (sampling_rate) {
        return sampling_rate / 1000 * Opus.getMaxFrameDuration();
    };
    return Opus;
})();
var OpusEncoder = (function () {
    function OpusEncoder(sampling_rate, channels, app, frame_duration) {
        if (frame_duration === void 0) { frame_duration = 20; }
        this.handle = 0;
        this.frame_size = 0;
        this.in_ptr = 0;
        this.in_off = 0;
        this.out_ptr = 0;
        if (!Opus.validFrameDuration(frame_duration))
            throw 'invalid frame duration';
        this.frame_size = sampling_rate * frame_duration / 1000;
        var err_ptr = allocate(4, 'i32', ALLOC_STACK);
        this.handle = _opus_encoder_create(sampling_rate, channels, app, err_ptr);
        if (getValue(err_ptr, 'i32') != 0 /* OK */)
            throw 'opus_encoder_create failed: ' + getValue(err_ptr, 'i32');
        this.in_ptr = _malloc(this.frame_size * channels * 4);
        this.in_len = this.frame_size * channels;
        this.in_i16 = HEAP16.subarray(this.in_ptr >> 1, (this.in_ptr >> 1) + this.in_len);
        this.in_f32 = HEAPF32.subarray(this.in_ptr >> 2, (this.in_ptr >> 2) + this.in_len);
        this.out_bytes = Opus.getMaxFrameSize();
        this.out_ptr = _malloc(this.out_bytes);
        this.out_buf = HEAPU8.subarray(this.out_ptr, this.out_ptr + this.out_bytes);
    }
    OpusEncoder.prototype.encode = function (pcm) {
        var output = [];
        var pcm_off = 0;
        while (pcm.length - pcm_off >= this.in_len - this.in_off) {
            if (this.in_off > 0) {
                this.in_i16.set(pcm.subarray(pcm_off, pcm_off + this.in_len - this.in_off), this.in_off);
                pcm_off += this.in_len - this.in_off;
                this.in_off = 0;
            }
            else {
                this.in_i16.set(pcm.subarray(pcm_off, pcm_off + this.in_len));
                pcm_off += this.in_len;
            }
            var ret = _opus_encode(this.handle, this.in_ptr, this.frame_size, this.out_ptr, this.out_bytes);
            if (ret <= 0)
                throw 'opus_encode failed: ' + ret;
            var packet = new ArrayBuffer(ret);
            new Uint8Array(packet).set(this.out_buf.subarray(0, ret));
            output.push(packet);
        }
        if (pcm_off < pcm.length) {
            this.in_i16.set(pcm.subarray(pcm_off));
            this.in_off = pcm.length - pcm_off;
        }
        return output;
    };
    OpusEncoder.prototype.encode_float = function (pcm) {
        var output = [];
        var pcm_off = 0;
        while (pcm.length - pcm_off >= this.in_len - this.in_off) {
            if (this.in_off > 0) {
                this.in_f32.set(pcm.subarray(pcm_off, pcm_off + this.in_len - this.in_off), this.in_off);
                pcm_off += this.in_len - this.in_off;
                this.in_off = 0;
            }
            else {
                this.in_f32.set(pcm.subarray(pcm_off, pcm_off + this.in_len));
                pcm_off += this.in_len;
            }
            var ret = _opus_encode_float(this.handle, this.in_ptr, this.frame_size, this.out_ptr, this.out_bytes);
            if (ret <= 0)
                throw 'opus_encode failed: ' + ret;
            var packet = new ArrayBuffer(ret);
            new Uint8Array(packet).set(this.out_buf.subarray(0, ret));
            output.push(packet);
        }
        if (pcm_off < pcm.length) {
            this.in_f32.set(pcm.subarray(pcm_off));
            this.in_off = pcm.length - pcm_off;
        }
        return output;
    };
    OpusEncoder.prototype.encode_final = function () {
        if (this.in_off == 0)
            return new ArrayBuffer(0);
        for (var i = this.in_off; i < this.in_len; ++i)
            this.in_i16[i] = 0;
        var ret = _opus_encode(this.handle, this.in_ptr, this.frame_size, this.out_ptr, this.out_bytes);
        if (ret <= 0)
            throw 'opus_encode failed: ' + ret;
        var packet = new ArrayBuffer(ret);
        new Uint8Array(packet).set(this.out_buf.subarray(0, ret));
        return packet;
    };
    OpusEncoder.prototype.encode_float_final = function () {
        if (this.in_off == 0)
            return new ArrayBuffer(0);
        for (var i = this.in_off; i < this.in_len; ++i)
            this.in_f32[i] = 0;
        var ret = _opus_encode_float(this.handle, this.in_ptr, this.frame_size, this.out_ptr, this.out_bytes);
        if (ret <= 0)
            throw 'opus_encode failed: ' + ret;
        var packet = new ArrayBuffer(ret);
        new Uint8Array(packet).set(this.out_buf.subarray(0, ret));
        return packet;
    };
    OpusEncoder.prototype.destroy = function () {
        if (!this.handle)
            return;
        _opus_encoder_destroy(this.handle);
        _free(this.in_ptr);
        this.handle = this.in_ptr = 0;
    };
    return OpusEncoder;
})();
var OpusDecoder = (function () {
    function OpusDecoder(sampling_rate, channels) {
        this.handle = 0;
        this.in_ptr = 0;
        this.out_ptr = 0;
        this.channels = channels;
        var err_ptr = allocate(4, 'i32', ALLOC_STACK);
        this.handle = _opus_decoder_create(sampling_rate, channels, err_ptr);
        if (getValue(err_ptr, 'i32') != 0 /* OK */)
            throw 'opus_decoder_create failed: ' + getValue(err_ptr, 'i32');
        this.in_ptr = _malloc(Opus.getMaxFrameSize(channels));
        this.in_buf = HEAPU8.subarray(this.in_ptr, this.in_ptr + Opus.getMaxFrameSize(channels));
        this.out_len = Opus.getMaxSamplesPerChannel(sampling_rate);
        var out_bytes = this.out_len * channels * 4;
        this.out_ptr = _malloc(out_bytes);
        this.out_i16 = HEAP16.subarray(this.out_ptr >> 1, (this.out_ptr + out_bytes) >> 1);
        this.out_f32 = HEAPF32.subarray(this.out_ptr >> 2, (this.out_ptr + out_bytes) >> 2);
    }
    OpusDecoder.prototype.decode = function (packet) {
        this.in_buf.set(new Uint8Array(packet));
        var ret = _opus_decode(this.handle, this.in_ptr, packet.byteLength, this.out_ptr, this.out_len, 0);
        if (ret < 0)
            throw 'opus_decode failed: ' + ret;
        var samples = new Int16Array(ret * this.channels);
        samples.set(this.out_i16.subarray(0, samples.length));
        return samples;
    };
    OpusDecoder.prototype.decode_float = function (packet) {
        this.in_buf.set(new Uint8Array(packet));
        var ret = _opus_decode_float(this.handle, this.in_ptr, packet.byteLength, this.out_ptr, this.out_len, 0);
        if (ret < 0)
            throw 'opus_decode failed: ' + ret;
        var samples = new Float32Array(ret * this.channels);
        samples.set(this.out_f32.subarray(0, samples.length));
        return samples;
    };
    OpusDecoder.prototype.destroy = function () {
        if (!this.handle)
            return;
        _opus_decoder_destroy(this.handle);
        _free(this.in_ptr);
        _free(this.out_ptr);
        this.handle = this.in_ptr = this.out_ptr = 0;
    };
    return OpusDecoder;
})();
