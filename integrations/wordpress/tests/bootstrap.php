<?php
/**
 * Loads the plugin's classes without WordPress: Brain Monkey stubs the
 * WordPress functions each test needs.
 *
 * @package ReputeCast
 */

require dirname( __DIR__ ) . '/vendor/autoload.php';
require __DIR__ . '/wp-classes.php';

define( 'ABSPATH', __DIR__ . '/' );
define( 'MINUTE_IN_SECONDS', 60 );
define( 'HOUR_IN_SECONDS', 3600 );

// The plugin's constants, from its main file, so they can't drift.
$reputecast_main = (string) file_get_contents( dirname( __DIR__ ) . '/reputecast/reputecast.php' );
preg_match( "/define\(\s*'REPUTECAST_VERSION',\s*'([^']+)'/", $reputecast_main, $reputecast_version );
define( 'REPUTECAST_VERSION', $reputecast_version[1] );
define( 'REPUTECAST_FILE', dirname( __DIR__ ) . '/reputecast/reputecast.php' );
define( 'REPUTECAST_DIR', dirname( __DIR__ ) . '/reputecast' );

foreach ( glob( REPUTECAST_DIR . '/includes/class-*.php' ) as $reputecast_file ) {
	require $reputecast_file;
}
