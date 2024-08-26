addEventListener('fetch', event => {
	event.respondWith(handleRequest(event.request))
})

// random_image_bak.go 使用这份
async function handleRequest(request) {
	const url = new URL(request.url)
	const path = url.pathname

	if (path === '/list') {
		return handleListRequest(request)
	} else if (path === '/img') {
		return handleImgRequest(request)
	} else {
		return new Response('Not Found', { status: 404 })
	}
}

async function handleListRequest(request) {
	const url = new URL(request.url)
	const searchParams = new URLSearchParams(url.search)

	const apiUrl = `https://wallhaven.cc/api/v1/search?${searchParams.toString()}`

	const response = await fetch(apiUrl, {
		method: 'GET',
		headers: {
			'Content-Type': 'application/json'
		}
	})

	if (!response.ok) {
		return new Response('Error fetching data from Wallhaven API', {
			status: response.status,
			statusText: response.statusText
		})
	}

	const data = await response.json()

	return new Response(JSON.stringify(data), {
		headers: { 'Content-Type': 'application/json' }
	})
}

async function handleImgRequest(request) {
	const url = new URL(request.url);
	const imagePath = url.searchParams.get('path');

	if (!imagePath) {
		return new Response('Image path is required', { status: 400 });
	}

	// 发起图片请求而不进行处理或缓存
	const response = await fetch(imagePath);

	if (!response.ok) {
		return new Response('Error fetching image', {
			status: response.status,
			statusText: response.statusText,
		});
	}

	// 将响应直接返回给客户端，不进行任何处理
	return new Response(response.body, {
		headers: {
			'Content-Type': response.headers.get('Content-Type'),
			'Cache-Control': 'no-store', // 如果不想缓存，设置为 no-store
		},
		status: response.status,
		statusText: response.statusText,
	});
}