module.exports = {
  configureWebpack: config => {
    if (process.env.NODE_ENV === 'production') {
      const CompressionPlugin = require('compression-webpack-plugin');
      const zlib = require('zlib');
      config.plugins.push(new CompressionPlugin({
        filename: '[path][base].br',
        algorithm: 'brotliCompress',
        test: /\.(js|css|html|svg|map)$/,
        compressionOptions: {
          params: {
            [zlib.constants.BROTLI_PARAM_QUALITY]: 11
          }
        },
        threshold: 10240,
        minRatio: 0.8
      }));
      config.plugins.push(new CompressionPlugin({
        filename: '[path][base].gz',
        algorithm: 'gzip',
        test: /\.(js|css|html|svg|map)$/,
        threshold: 10240,
        minRatio: 0.8
      }));
    }
  }
};
