<template>
  <!--
    新闻 / 报纸图标（Element Plus 图标包里没有 NewsPaper）
    1024×1024 网格，与官方图标一致；currentColor 跟随文字颜色
    纸张外框 + 报头 + 正文行 + 图片块，小尺寸下也能辨认
  -->
  <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
    <!-- 纸张外框：用 evenodd 挖空内部，得到一圈边框 -->
    <path
      fill="currentColor"
      fill-rule="evenodd"
      d="M128 96h768a32 32 0 0 1 32 32v768a32 32 0 0 1-32 32H128a32 32 0 0 1-32-32V128a32 32 0 0 1 32-32Zm32 64v704h704V160H160Z"
    />
    <!-- 报头标题 -->
    <rect fill="currentColor" x="192" y="224" width="352" height="48" rx="8" />
    <!-- 报头右侧日期/期号 -->
    <rect fill="currentColor" x="672" y="224" width="160" height="48" rx="8" />
    <!-- 标题下分隔线 -->
    <rect fill="currentColor" x="192" y="320" width="640" height="16" rx="8" />
    <!-- 左栏正文行 -->
    <rect fill="currentColor" x="192" y="384" width="256" height="32" rx="8" />
    <rect fill="currentColor" x="192" y="448" width="256" height="32" rx="8" />
    <rect fill="currentColor" x="192" y="512" width="256" height="32" rx="8" />
    <rect fill="currentColor" x="192" y="576" width="256" height="32" rx="8" />
    <!-- 右栏配图区块 -->
    <rect fill="currentColor" x="512" y="384" width="320" height="288" rx="12" />
    <!-- 底部通栏文字 -->
    <rect fill="currentColor" x="192" y="720" width="640" height="32" rx="8" />
    <rect fill="currentColor" x="192" y="784" width="448" height="32" rx="8" />
  </svg>
</template>
