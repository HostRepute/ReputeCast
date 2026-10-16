import { createRequire } from 'node:module';

import { defineConfig } from 'vitest/config';

const require = createRequire(import.meta.url);

export default defineConfig({
	resolve: {
		alias: {
			// The CommonJS build, which n8n loads community nodes with.
			'n8n-workflow': require.resolve('n8n-workflow'),
		},
	},
	test: {
		include: ['test/**/*.test.mts'],
		environment: 'node',
	},
});
