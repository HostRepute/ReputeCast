<?php
/**
 * Parses every PHP file of the plugin with the running PHP, so a file that
 * only an older or newer PHP can read fails here (CI runs PHP 8.1 and 8.4).
 *
 * Usage: php scripts/lint.php
 *
 * @package ReputeCast
 */

$root   = dirname( __DIR__ );
$failed = 0;
$count  = 0;
$files  = new RecursiveIteratorIterator( new RecursiveDirectoryIterator( $root . '/reputecast', FilesystemIterator::SKIP_DOTS ) );
foreach ( $files as $file ) {
	if ( 'php' !== $file->getExtension() ) {
		continue;
	}
	++$count;
	try {
		token_get_all( (string) file_get_contents( $file->getPathname() ), TOKEN_PARSE );
	} catch ( ParseError $e ) {
		++$failed;
		fwrite( STDERR, substr( $file->getPathname(), strlen( $root ) + 1 ) . ':' . $e->getLine() . ': ' . $e->getMessage() . PHP_EOL );
	}
}
echo 'PHP ' . PHP_VERSION . ': ' . $count . ' files parsed, ' . $failed . ' failed.' . PHP_EOL;
exit( $failed > 0 ? 1 : 0 );
