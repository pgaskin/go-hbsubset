#include <cstdint>
#include <cstddef>

#include <hb.h>
#include <hb-subset.h>

__attribute__((visibility("default")))
extern "C" hb_face_t *hbw_face_create(const char *data, unsigned int length, unsigned int index) {
	hb_blob_t *blob = hb_blob_create_or_fail(data, length, HB_MEMORY_MODE_WRITABLE, nullptr, nullptr);
	if (!blob) {
		return nullptr;
	}
	hb_face_t *face = hb_face_create_or_fail(blob, index);
	hb_blob_destroy(blob);
	return face;
}

__attribute__((visibility("default")))
extern "C" unsigned int hbw_face_count(const char *data, unsigned int length) {
	hb_blob_t *blob = hb_blob_create_or_fail(data, length, HB_MEMORY_MODE_WRITABLE, nullptr, nullptr);
	if (!blob) {
		return 0;
	}
	unsigned int n = hb_face_count(blob);
	hb_blob_destroy(blob);
	return n;
}

extern "C" struct hbw_bytes {
	const char *ptr;
	unsigned int len;
	hb_blob_t *handle;
};

__attribute__((visibility("default")))
extern "C" struct hbw_bytes hbw_blob_data(hb_blob_t *blob) {
	if (!blob) {
		return (hbw_bytes){};
	}
	unsigned int length = 0;
	const char *data = hb_blob_get_data(blob, &length);
	return (hbw_bytes){
		.ptr = data,
		.len = length,
		.handle = blob,
	};
}

__attribute__((visibility("default")))
extern "C" struct hbw_bytes hbw_face_blob(hb_face_t *face) {
	return hbw_blob_data(hb_face_reference_blob(face));
}

__attribute__((visibility("default")))
extern "C" struct hbw_bytes hbw_face_table(hb_face_t *face, hb_tag_t tag) {
	return hbw_blob_data(hb_face_reference_table(face, tag));
}

__attribute__((visibility("default")))
extern "C" unsigned int hbw_face_table_tags(hb_face_t *face, unsigned int start, hb_tag_t *out, unsigned int cap) {
	unsigned int count = cap;
	hb_face_get_table_tags(face, start, &count, out);
	return count;
}

__attribute__((visibility("default")))
extern "C" unsigned int hbw_map_entries(const hb_map_t *map, hb_codepoint_t *keys, hb_codepoint_t *vals, unsigned int cap) {
	int idx = -1;
	hb_codepoint_t key, val;
	unsigned int n = 0;
	while (n < cap && hb_map_next(map, &idx, &key, &val)) {
		keys[n] = key;
		vals[n] = val;
		n++;
	}
	return n;
}

// no-op fd_write (referenced by stdio and abort)
extern "C" int32_t __imported_wasi_snapshot_preview1_fd_write(int32_t, int32_t, int32_t, int32_t nwritten) {
	*(uint32_t *)(uintptr_t)nwritten = 0;
	return 0;
}

// no-op fd_seek (referenced by stdio and abort)
extern "C" int32_t __imported_wasi_snapshot_preview1_fd_seek(int32_t, int64_t, int32_t, int32_t newoffset) {
	*(int64_t *)(uintptr_t)newoffset = 0;
	return 0;
}

// no-op fd_close (referenced by stdio and abort)
extern "C" int32_t __imported_wasi_snapshot_preview1_fd_close(int32_t) {
	return 0;
}

// no-op proc_exit (referenced by abort)
extern "C" __attribute__((noreturn)) void __imported_wasi_snapshot_preview1_proc_exit(int32_t) {
	__builtin_trap();
}

// no-op environ_sizes_get (referenced by getenv)
extern "C" int32_t __imported_wasi_snapshot_preview1_environ_sizes_get(int32_t count, int32_t buf_size) {
	*(uint32_t *)(uintptr_t)count = 0;
	*(uint32_t *)(uintptr_t)buf_size = 0;
	return 0;
}

// no-op environ_get (referenced by getenv)
extern "C" int32_t __imported_wasi_snapshot_preview1_environ_get(int32_t, int32_t) {
	return 0;
}

// no pre-opened file descriptors
extern "C" int32_t __imported_wasi_snapshot_preview1_fd_prestat_get(int32_t, int32_t) {
	return 8; // EBADF
}

// no pre-opened file descriptors
extern "C" int32_t __imported_wasi_snapshot_preview1_fd_prestat_dir_name(int32_t, int32_t, int32_t) {
	return 8; // EBADF
}
