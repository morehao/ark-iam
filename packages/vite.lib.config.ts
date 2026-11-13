import { resolve } from 'node:path'

// @ark-iam/{ui,auth,api,types} 共用的库构建配置生成器（返回普通对象，不 import vite，
// 以便在未把 vite 链接进各包 node_modules 的 workspace 下也能用任意 vite 可执行文件加载本配置文件）。
//
// 设计要点：
// - 只产 ESM 单文件（dist/index.js）+ sourcemap；类型声明由各包 build 脚本里的
//   `tsc -p tsconfig.build.json`（emitDeclarationOnly）单独产出 dist/index.d.ts。
// - 所有第三方依赖 + 跨包 @ark-iam/* 一律 external（由消费者安装，见 peerDependencies），
//   避免把 react / antd / 彼此的源码重复打进产物。
// - 仓内 app 通过 vite resolve.alias 走 src（dev 免 build），只有发布 npm 包时才用本产物。
export interface LibBuildOptions {
  /** UMD/global 名，仅供非 ESM 场景命名（本仓只产 ESM，占位用）。 */
  name: string
  /** 入口相对包根路径，默认 src/index.ts。 */
  entry?: string
  /** 额外 external 的模块（跨包 @ark-iam/* 会自动外置）。 */
  external?: string[]
}

export function defineLibConfig(opts: LibBuildOptions) {
  return {
    build: {
      lib: {
        entry: resolve(process.cwd(), opts.entry ?? 'src/index.ts'),
        name: opts.name,
        formats: ['es'] as const,
        fileName: () => 'index.js',
      },
      rollupOptions: {
        external: (id: string) =>
          id.startsWith('@ark-iam/') ||
          (opts.external ?? []).some((e) => id === e || id.startsWith(e + '/')) ||
          (!id.startsWith('.') && !id.startsWith('/')),
      },
      sourcemap: true,
      minify: false,
      outDir: 'dist',
      emptyOutDir: true,
    },
  }
}
