import { error, json, type RequestHandler } from '@sveltejs/kit';
import { PutObjectCommand, S3Client } from '@aws-sdk/client-s3';
import { randomBytes } from 'crypto';
import { env } from '$env/dynamic/private';
import sharp from 'sharp';

const s3 = new S3Client({
	region: env.STORAGE_S3_REGION,
	endpoint: env.STORAGE_S3_ENDPOINT,
	credentials: {
		accessKeyId: env.STORAGE_S3_ACCESS_KEY_ID,
		secretAccessKey: env.STORAGE_S3_SECRET_ACCESS_KEY
	}
});

export const POST: RequestHandler = async ({ request, locals }) => {
	if (!locals.session) {
		throw error(401, 'Authentication required');
	}

	const payload = Object.fromEntries(await request.formData());

	const { file } = payload as {
		file: File;
	};

	// 2025/09/6d2be8edb5g7_preview.jpg
	// 2025/09/6d2be8edb5g7.jpg

	const now = new Date();
	const year = now.getFullYear();
	const month = ('0' + (now.getMonth() + 1)).slice(-2);
	const randomString = randomBytes(6).toString('hex');

	// Process the uploaded image
	const imageBuffer = Buffer.from(await file.arrayBuffer());

	// Compress the main image (mozjpeg 85)
	const compressedImage = await sharp(imageBuffer, { autoOrient: true })
		.jpeg({
			quality: 85,
			mozjpeg: true
		})
		.toBuffer();

	// Create preview version (mozjpeg 80, max width 1200, no upscaling)
	const previewImage = await sharp(imageBuffer, { autoOrient: true })
		.resize(1200, null, {
			withoutEnlargement: true
		})
		.jpeg({
			quality: 80,
			mozjpeg: true
		})
		.toBuffer();

	// Get preview image metadata
	const previewMetadata = await sharp(previewImage).metadata();

	const mainImageKey = `${year}/${month}/${randomString}.jpg`;
	const previewImageKey = `${year}/${month}/${randomString}_preview.jpg`;

	// Upload compressed main image
	await s3.send(
		new PutObjectCommand({
			Bucket: env.STORAGE_S3_BUCKET,
			Key: mainImageKey,
			Body: compressedImage,
			ContentType: 'image/jpeg',
			Metadata: {
				original: file.name
			}
		})
	);

	// Upload preview image
	await s3.send(
		new PutObjectCommand({
			Bucket: env.STORAGE_S3_BUCKET,
			Key: previewImageKey,
			Body: previewImage,
			ContentType: 'image/jpeg'
		})
	);

	return json({
		main: mainImageKey,
		preview: previewImageKey,
		width: previewMetadata.width,
		height: previewMetadata.height
	});
};
