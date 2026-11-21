// sharp's prebuilt libvips has no ImageMagick support, but govips calls its
// loader for formats like BMP, so the program wouldn't link. This stub makes
// those formats fail to load instead. It declares what it needs rather than
// including vips.h, so that the package doesn't add libvips's linker flags a
// second time.
#include <stddef.h>

void vips_error(const char *domain, const char *fmt, ...);

int vips_magickload_buffer(void *buf, size_t len, void **out, ...) {
	vips_error("magickload_buffer", "libvips is built without ImageMagick");
	return -1;
}
