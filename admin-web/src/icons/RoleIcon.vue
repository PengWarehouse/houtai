<template>
  <!--
    角色图标（Element Plus 图标包里没有）
    1024×1024 网格，与官方图标一致；currentColor 跟随文字颜色
    盾牌实心 + 位运算镂空出人物剪影，小尺寸下也能看清轮廓
  -->
  <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
    <path
      fill="currentColor"
      fill-rule="evenodd"
      d="M512 96 896 240v240c0 220-166 400-384 464-218-64-384-244-384-464V240L512 96Z
         M512 284a108 108 0 1 1 0 216 108 108 0 0 1 0-216Z
         M512 528c-104 0-188 72-188 160v24h376v-24c0-88-84-160-188-160Z"
    />
  </svg>
</template>
