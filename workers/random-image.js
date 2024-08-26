addEventListener('fetch', event => {
    event.respondWith(handleRequest(event.request))
})

const CACHE = new Map(); // 用于缓存API数据和索引

async function handleRequest(request) {
    const url = new URL(request.url)
    const path = url.pathname

    if (path === '/img') {
        return handleImgRequest(request)
    } else {
        return new Response('Not Found', { status: 404 })
    }
}

async function fetchWallhavenData(searchParams) {
    const apiUrl = `https://wallhaven.cc/api/v1/search?${searchParams.toString()}`

    const response = await fetch(apiUrl, {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json'
        }
    })

    if (!response.ok) {
        throw new Error('Error fetching data from Wallhaven API')
    }

    const data = await response.json()
    return data.data; // 直接返回图片数据数组
}

async function handleImgRequest(request) {
    const url = new URL(request.url);
    const searchParams = new URLSearchParams(url.search);
    const requestKey = searchParams.toString(); // 用于标识唯一请求的键值

    if (!CACHE.has(requestKey)) {
        try {
            const images = await fetchWallhavenData(searchParams);
            CACHE.set(requestKey, { images, index: 0 });
        } catch (error) {
            return new Response('Error fetching data from Wallhaven API', {
                status: 500,
                statusText: error.message,
            });
        }
    }

    const cacheEntry = CACHE.get(requestKey);
    const images = cacheEntry.images;
    let imageIndex = cacheEntry.index; // 当前要返回的图片索引

    if (imageIndex >= images.length) {
        imageIndex = 0; // 重置索引以便重新轮询
    }

    const imageUrl = images[imageIndex].path; // 获取当前索引对应的图片路径
    cacheEntry.index = imageIndex + 1; // 更新缓存中的索引

    const imageResponse = await fetch(imageUrl);

    if (!imageResponse.ok) {
        return new Response('Error fetching image', {
            status: imageResponse.status,
            statusText: imageResponse.statusText,
        });
    }

    return new Response(imageResponse.body, {
        headers: {
            'Content-Type': imageResponse.headers.get('Content-Type'),
            'Cache-Control': 'no-store', // 如果不想缓存，设置为 no-store
        },
        status: imageResponse.status,
        statusText: imageResponse.statusText,
    });
}
