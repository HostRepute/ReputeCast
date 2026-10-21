<?php
/**
 * Plugin Name:       ReputeCast
 * Plugin URI:        https://github.com/hostrepute/reputecast/tree/main/integrations/wordpress
 * Description:       Signup forms from ReputeCast, the open-source newsletter platform, as a block and a shortcode. Published posts become campaign drafts in ReputeCast, the site reports its health there, and WooCommerce stores get the checkout's marketing consent and abandoned checkouts.
 * Version:           1.4.0
 * Requires at least: 6.4
 * Requires PHP:      8.1
 * Author:            HostRepute
 * Author URI:        https://hostrepute.com
 * License:           GPL-2.0-or-later
 * License URI:       https://www.gnu.org/licenses/gpl-2.0.html
 * Text Domain:       reputecast
 * Domain Path:       /languages
 *
 * @package ReputeCast
 */

defined( 'ABSPATH' ) || exit;

define( 'REPUTECAST_VERSION', '1.4.0' );
define( 'REPUTECAST_FILE', __FILE__ );
define( 'REPUTECAST_DIR', __DIR__ );

require_once __DIR__ . '/includes/class-options.php';
require_once __DIR__ . '/includes/class-client.php';
require_once __DIR__ . '/includes/class-languages.php';
require_once __DIR__ . '/includes/class-form-usage.php';
require_once __DIR__ . '/includes/class-reporter.php';
require_once __DIR__ . '/includes/class-forms.php';
require_once __DIR__ . '/includes/class-posts.php';
require_once __DIR__ . '/includes/class-status.php';
require_once __DIR__ . '/includes/class-settings-page.php';
require_once __DIR__ . '/includes/class-woocommerce.php';
require_once __DIR__ . '/includes/class-plugin.php';

register_activation_hook( __FILE__, array( 'ReputeCast\\Plugin', 'activate' ) );
register_deactivation_hook( __FILE__, array( 'ReputeCast\\Plugin', 'deactivate' ) );

ReputeCast\Plugin::boot();
