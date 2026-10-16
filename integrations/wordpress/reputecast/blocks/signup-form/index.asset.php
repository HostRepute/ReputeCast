<?php
/**
 * The editor script's dependencies (the script is plain JavaScript: no build step).
 *
 * @package ReputeCast
 */

defined( 'ABSPATH' ) || exit;

return array(
	'dependencies' => array( 'wp-api-fetch', 'wp-block-editor', 'wp-blocks', 'wp-components', 'wp-element', 'wp-i18n' ),
	'version'      => '1.3.0',
);
