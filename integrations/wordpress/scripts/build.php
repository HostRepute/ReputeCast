<?php
/**
 * Builds the plugin's zip, as the WordPress plugin directory and
 * Plugins › Add New › Upload take it: dist/reputecast-{version}.zip, with
 * the reputecast/ folder inside. Fails when the version numbers disagree.
 *
 * Usage: php scripts/build.php
 *
 * @package ReputeCast
 */

$root       = dirname( __DIR__ );
$plugin_dir = $root . '/reputecast';

/**
 * Stops with a message.
 *
 * @param string $message What went wrong.
 */
function reputecast_fail( string $message ): void {
	fwrite( STDERR, $message . PHP_EOL );
	exit( 1 );
}

$main     = (string) file_get_contents( $plugin_dir . '/reputecast.php' );
$readme   = (string) file_get_contents( $plugin_dir . '/readme.txt' );
$block    = json_decode( (string) file_get_contents( $plugin_dir . '/blocks/signup-form/block.json' ), true );
$asset    = (string) file_get_contents( $plugin_dir . '/blocks/signup-form/index.asset.php' );
$versions = array(
	'reputecast.php header'   => preg_match( '/^\s*\*\s*Version:\s*(\S+)/m', $main, $m ) ? $m[1] : '',
	'REPUTECAST_VERSION'      => preg_match( "/define\(\s*'REPUTECAST_VERSION',\s*'([^']+)'/", $main, $m ) ? $m[1] : '',
	'readme.txt Stable tag'   => preg_match( '/^Stable tag:\s*(\S+)/m', $readme, $m ) ? $m[1] : '',
	'block.json version'      => is_array( $block ) ? (string) ( $block['version'] ?? '' ) : '',
	'index.asset.php version' => preg_match( "/'version'\s*=>\s*'([^']+)'/", $asset, $m ) ? $m[1] : '',
);
$version  = $versions['reputecast.php header'];
foreach ( $versions as $where => $v ) {
	if ( $v !== $version || '' === $v ) {
		reputecast_fail( 'Version mismatch: ' . json_encode( $versions, JSON_PRETTY_PRINT ) );
	}
}
if ( ! class_exists( ZipArchive::class ) ) {
	reputecast_fail( 'The zip extension is missing.' );
}

$dist = $root . '/dist';
if ( ! is_dir( $dist ) && ! mkdir( $dist, 0755, true ) ) {
	reputecast_fail( 'Can’t create ' . $dist );
}
$target = $dist . '/reputecast-' . $version . '.zip';
if ( file_exists( $target ) ) {
	unlink( $target );
}
$zip = new ZipArchive();
if ( true !== $zip->open( $target, ZipArchive::CREATE ) ) {
	reputecast_fail( 'Can’t write ' . $target );
}

$files = array();
$walk  = new RecursiveIteratorIterator( new RecursiveDirectoryIterator( $plugin_dir, FilesystemIterator::SKIP_DOTS ) );
foreach ( $walk as $file ) {
	$relative = substr( $file->getPathname(), strlen( $plugin_dir ) + 1 );
	if ( str_starts_with( basename( $relative ), '.' ) ) {
		continue;
	}
	$files[] = $relative;
}
sort( $files );
// The same files make the same zip: a fixed time for every entry.
$mtime = 1767225600; // 2026-01-01.
foreach ( $files as $relative ) {
	$name = 'reputecast/' . str_replace( DIRECTORY_SEPARATOR, '/', $relative );
	$zip->addFile( $plugin_dir . '/' . $relative, $name );
	$zip->setMtimeName( $name, $mtime );
}
$zip->close();
echo 'Built ' . substr( $target, strlen( $root ) + 1 ) . ' (' . count( $files ) . ' files, ' . round( filesize( $target ) / 1024, 1 ) . ' KB).' . PHP_EOL;
