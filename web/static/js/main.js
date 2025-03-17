document.addEventListener('DOMContentLoaded', () => {
    const urlInput = document.getElementById('url-input');
    const proxyBtn = document.getElementById('proxy-btn');
    const resultContainer = document.getElementById('result-container');
    const proxyUrlInput = document.getElementById('proxy-url');
    const copyBtn = document.getElementById('copy-btn');
    const directLink = document.getElementById('direct-link');
    
    // 生成代理链接
    proxyBtn.addEventListener('click', () => {
        let targetUrl = urlInput.value.trim();
        if (!targetUrl) {
            alert('请输入有效的URL');
            return;
        }
        
        // 修复常见URL格式问题
        if (targetUrl.startsWith('http:/') && !targetUrl.startsWith('http://')) {
            targetUrl = 'http://' + targetUrl.substring(6);
        }
        if (targetUrl.startsWith('https:/') && !targetUrl.startsWith('https://')) {
            targetUrl = 'https://' + targetUrl.substring(7);
        }
        
        // 检查URL格式
        try {
            new URL(targetUrl);
        } catch {
            alert('请输入有效的URL，包含http://或https://');
            return;
        }
        
        // 构建代理URL
        const proxyUrl = `/proxy/${targetUrl}`;
        proxyUrlInput.value = window.location.origin + proxyUrl;
        directLink.href = proxyUrl;
        
        // 显示结果
        resultContainer.style.display = 'block';
        
        // 滚动到结果区域
        resultContainer.scrollIntoView({ behavior: 'smooth' });
    });
    
    // 复制链接到剪贴板
    copyBtn.addEventListener('click', () => {
        proxyUrlInput.select();
        document.execCommand('copy');
        
        // 显示复制成功提示
        const originalText = copyBtn.innerHTML;
        copyBtn.innerHTML = '<span class="material-symbols-rounded">check</span>';
        setTimeout(() => {
            copyBtn.innerHTML = originalText;
        }, 2000);
    });
}); 