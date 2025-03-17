document.addEventListener('DOMContentLoaded', () => {
    // 获取DOM元素
    const urlInput = document.getElementById('url-input');
    const proxyBtn = document.getElementById('proxy-btn');
    const resultContainer = document.getElementById('result-container');
    const proxyUrlInput = document.getElementById('proxy-url');
    const copyBtn = document.getElementById('copy-btn');
    const directLink = document.getElementById('direct-link');
    const themeToggle = document.getElementById('theme-toggle');
    const tiltCard = document.querySelector('.tilt-card');
    
    // 主题切换功能 - 增加基于时间的自动切换
    function setThemeBasedOnTime() {
        const currentHour = new Date().getHours();
        // 晚上8点到早上6点使用暗色模式
        return (currentHour >= 20 || currentHour < 6);
    }
    
    // 设置主题
    function setTheme(isDark) {
        const themeIndicator = document.getElementById('theme-indicator');
        const isAuto = localStorage.getItem('autoTheme') === 'true';
        
        if (isDark) {
            document.body.classList.add('dark-mode');
            themeToggle.innerHTML = '<span class="material-symbols-rounded">light_mode</span>';
            localStorage.setItem('theme', 'dark');
            themeIndicator.textContent = isAuto ? '自动：夜间模式' : '手动：夜间模式';
        } else {
            document.body.classList.remove('dark-mode');
            themeToggle.innerHTML = '<span class="material-symbols-rounded">dark_mode</span>';
            localStorage.setItem('theme', 'light');
            themeIndicator.textContent = isAuto ? '自动：日间模式' : '手动：日间模式';
        }
    }
    
    // 初始化主题
    function initTheme() {
        const savedTheme = localStorage.getItem('theme');
        const savedAutoTheme = localStorage.getItem('autoTheme');
        
        if (savedTheme === null && (savedAutoTheme === null || savedAutoTheme === 'true')) {
            // 如果没有保存的主题或自动主题设置为true，则根据时间自动设置
            setTheme(setThemeBasedOnTime());
            localStorage.setItem('autoTheme', 'true');
        } else if (savedTheme === 'dark') {
            // 如果有保存的主题，则使用保存的主题
            setTheme(true);
        } else {
            setTheme(false);
        }
    }
    
    // 初始化主题
    initTheme();
    
    // 如果启用了自动主题，每小时检查一次
    setInterval(() => {
        if (localStorage.getItem('autoTheme') === 'true') {
            setTheme(setThemeBasedOnTime());
        }
    }, 60 * 60 * 1000); // 每小时检查一次
    
    // 主题切换按钮点击事件
    themeToggle.addEventListener('click', (e) => {
        // 按住Shift键点击切换自动主题模式
        if (e.shiftKey) {
            const isAutoTheme = localStorage.getItem('autoTheme') === 'true';
            localStorage.setItem('autoTheme', isAutoTheme ? 'false' : 'true');
            
            if (!isAutoTheme) {
                // 如果开启自动主题，立即应用基于时间的主题
                setTheme(setThemeBasedOnTime());
                alert('已开启自动主题模式（基于时间）');
            } else {
                alert('已关闭自动主题模式');
            }
            return;
        }
        
        // 普通点击切换主题
        const isDarkMode = document.body.classList.contains('dark-mode');
        setTheme(!isDarkMode);
        
        // 关闭自动主题
        localStorage.setItem('autoTheme', 'false');
    });
    
    // 添加主题切换提示
    themeToggle.setAttribute('title', '点击切换主题 | Shift+点击切换自动主题模式');
    
    // 3D倾斜效果
    if (window.innerWidth > 768) {
        tiltCard.addEventListener('mousemove', (e) => {
            const card = e.currentTarget;
            const cardRect = card.getBoundingClientRect();
            const cardCenterX = cardRect.left + cardRect.width / 2;
            const cardCenterY = cardRect.top + cardRect.height / 2;
            
            // 计算鼠标位置相对于卡片中心的偏移
            const offsetX = (e.clientX - cardCenterX) / (cardRect.width / 2);
            const offsetY = (e.clientY - cardCenterY) / (cardRect.height / 2);
            
            // 应用3D旋转效果，最大旋转角度为5度
            card.style.transform = `perspective(1000px) rotateX(${-offsetY * 5}deg) rotateY(${offsetX * 5}deg)`;
        });
        
        tiltCard.addEventListener('mouseleave', () => {
            // 鼠标离开时恢复原状
            tiltCard.style.transform = 'perspective(1000px) rotateX(0) rotateY(0)';
        });
    }
    
    // 动态模糊效果
    window.addEventListener('scroll', () => {
        const scrollPosition = window.scrollY;
        const maxBlur = 20; // 最大模糊值
        const blurValue = Math.min(10 + scrollPosition / 100, maxBlur);
        
        document.querySelectorAll('.glass-card').forEach(card => {
            card.style.backdropFilter = `blur(${blurValue}px)`;
            card.style.webkitBackdropFilter = `blur(${blurValue}px)`;
        });
    });
    
    // 生成代理链接
    proxyBtn.addEventListener('click', () => {
        // 显示加载状态
        proxyBtn.innerHTML = '<span class="material-symbols-rounded">hourglass_empty</span>处理中...';
        proxyBtn.disabled = true;
        
        setTimeout(() => {
            try {
                let targetUrl = urlInput.value.trim();
                if (!targetUrl) {
                    throw new Error('请输入有效的URL');
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
                } catch (e) {
                    throw new Error('请输入有效的URL，包含http://或https://');
                }
                
                // 从URL中提取文件名
                const urlPath = new URL(targetUrl).pathname;
                const fileName = urlPath.substring(urlPath.lastIndexOf('/') + 1);
                
                // 构建代理URL
                const proxyUrl = `${window.location.origin}/${targetUrl}`;
                proxyUrlInput.value = proxyUrl;
                directLink.href = `/${targetUrl}`;
                directLink.setAttribute("download", fileName);
                directLink.setAttribute("target", "_self");
                
                // 显示结果并平滑滚动
                resultContainer.style.display = 'block';
                resultContainer.scrollIntoView({ behavior: 'smooth' });
                
            } catch (err) {
                alert(err.message);
                console.error('生成代理链接时发生错误:', err);
            } finally {
                // 恢复按钮状态
                proxyBtn.innerHTML = '<span class="material-symbols-rounded">cloud_download</span>生成代理链接';
                proxyBtn.disabled = false;
            }
        }, 800); // 模拟处理延迟
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
    
    // 添加输入框回车触发生成
    urlInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            proxyBtn.click();
        }
    });
}); 