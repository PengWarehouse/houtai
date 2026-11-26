<script setup>
import { useRouter } from 'vue-router'
import {getToken} from "@/utils/auth"
const router = useRouter()

const homeBack=()=>{
if(getToken()){
  router.replace('/')
}else{
  router.push('/login')
}
}
</script>

<template>
  <div class="not-found">
    <el-result icon="warning" title="404" sub-title="页面不存在，或者你没有访问权限">
      <template #extra>
        <el-button type="primary" @click="homeBack">回到首页</el-button>
      </template>
    </el-result>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
</style>
