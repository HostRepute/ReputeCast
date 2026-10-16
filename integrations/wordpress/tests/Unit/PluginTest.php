<?php
/**
 * The plugin's hooks, files and version numbers.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Actions;
use Brain\Monkey\Functions;
use PHPUnit\Framework\Attributes\PreserveGlobalState;
use PHPUnit\Framework\Attributes\RunInSeparateProcess;
use ReputeCast\Forms;
use ReputeCast\Plugin;
use ReputeCast\Posts;
use ReputeCast\Reporter;
use ReputeCast\Settings_Page;
use ReputeCast\Status;
use ReputeCast\Tests\TestCase;

final class PluginTest extends TestCase {

	public function test_boot_hooks_in(): void {
		Functions\when( 'is_admin' )->justReturn( false );
		Actions\expectAdded( 'init' )->once()->with( array( Forms::class, 'register' ) );
		Actions\expectAdded( 'rest_api_init' )->twice();
		Actions\expectAdded( Reporter::CRON_HOOK )->once()->with( array( Reporter::class, 'run' ) );
		Actions\expectAdded( 'transition_post_status' )->once()->with( array( Posts::class, 'transition' ), 10, 3 );
		Actions\expectAdded( Posts::HOOK )->once()->with( array( Posts::class, 'send' ) );
		Actions\expectAdded( 'admin_menu' )->never();
		Plugin::boot();
	}

	public function test_the_settings_page_only_in_the_admin(): void {
		Functions\when( 'is_admin' )->justReturn( true );
		Functions\when( 'plugin_basename' )->justReturn( 'reputecast/reputecast.php' );
		Actions\expectAdded( 'admin_menu' )->once()->with( array( Settings_Page::class, 'menu' ) );
		Actions\expectAdded( 'admin_post_reputecast_connect' )->once();
		Actions\expectAdded( 'admin_post_reputecast_disconnect' )->once();
		Actions\expectAdded( 'admin_post_reputecast_report' )->once();
		Plugin::boot();
		$this->assertTrue( has_filter( 'plugin_action_links_reputecast/reputecast.php', array( Settings_Page::class, 'action_links' ) ) !== false );
	}

	public function test_activation_and_deactivation(): void {
		Plugin::activate();
		$this->assertSame( array(), $this->scheduled, 'nothing to report before Connect' );
		$this->connect_site();
		Plugin::activate();
		$this->assertSame( Reporter::CRON_HOOK, $this->scheduled[0]['hook'] );
		Plugin::deactivate();
		$this->assertSame( array(), $this->scheduled );
		$this->assertTrue( \ReputeCast\Options::is_connected(), 'the connection stays for when it is back' );
	}

	public function test_the_routes(): void {
		$routes = array();
		Functions\when( 'register_rest_route' )->alias(
			function ( $ns, $route ) use ( &$routes ) {
				$routes[] = $ns . $route;
				return true;
			}
		);
		Forms::register_routes();
		Status::register_routes();
		$this->assertSame( array( 'reputecast/v1/forms', 'reputecast/v1/status' ), $routes );
	}

	#[RunInSeparateProcess]
	#[PreserveGlobalState( false )]
	public function test_uninstalling_removes_what_the_plugin_kept(): void {
		$this->connect_site();
		$deleted = array();
		Functions\when( 'delete_post_meta_by_key' )->alias(
			static function ( $key ) use ( &$deleted ) {
				$deleted[] = $key;
				return true;
			}
		);
		Reporter::schedule();
		define( 'WP_UNINSTALL_PLUGIN', 'reputecast/reputecast.php' );
		require REPUTECAST_DIR . '/uninstall.php';
		$this->assertSame( array(), $this->options );
		$this->assertSame( array(), $this->scheduled );
		$this->assertSame( array( Posts::META ), $deleted );
	}

	public function test_the_version_numbers_agree(): void {
		$main   = (string) file_get_contents( REPUTECAST_FILE );
		$readme = (string) file_get_contents( REPUTECAST_DIR . '/readme.txt' );
		$block  = json_decode( (string) file_get_contents( REPUTECAST_DIR . '/blocks/signup-form/block.json' ), true );
		$asset  = (string) file_get_contents( REPUTECAST_DIR . '/blocks/signup-form/index.asset.php' );
		preg_match( '/^\s*\*\s*Version:\s*(\S+)/m', $main, $header );
		preg_match( '/^Stable tag:\s*(\S+)/m', $readme, $stable );
		preg_match( "/'version'\s*=>\s*'([^']+)'/", $asset, $asset_version );
		$this->assertSame( REPUTECAST_VERSION, $header[1] );
		$this->assertSame( REPUTECAST_VERSION, $stable[1] );
		$this->assertSame( REPUTECAST_VERSION, $block['version'] );
		$this->assertSame( REPUTECAST_VERSION, $asset_version[1] );
		// ReputeCast's call log and "Update available" know this version.
		$go = (string) file_get_contents( dirname( REPUTECAST_DIR, 3 ) . '/internal/integrations/wordpress.go' );
		$this->assertStringContainsString( 'const KnownPluginVersion = "' . REPUTECAST_VERSION . '"', $go );
		$readme_test = $this->header_value( $readme, 'Requires at least' );
		$this->assertSame( $this->header_value( $main, 'Requires at least' ), $readme_test );
		$this->assertSame( $this->header_value( $main, 'Requires PHP' ), $this->header_value( $readme, 'Requires PHP' ) );
		$this->assertCount( 5, array_map( 'trim', explode( ',', $this->header_value( $readme, 'Tags' ) ) ), 'the plugin directory takes five tags' );
	}

	public function test_the_block(): void {
		$block = json_decode( (string) file_get_contents( REPUTECAST_DIR . '/blocks/signup-form/block.json' ), true );
		$this->assertSame( 3, $block['apiVersion'] );
		$this->assertSame( Forms::BLOCK, $block['name'] );
		$this->assertSame( 'reputecast', $block['textdomain'] );
		$this->assertSame( array( 'formId', 'language' ), array_keys( $block['attributes'] ) );
		foreach ( array( 'editorScript', 'editorStyle', 'render' ) as $field ) {
			$this->assertFileExists( REPUTECAST_DIR . '/blocks/signup-form/' . substr( $block[ $field ], strlen( 'file:./' ) ) );
		}
		$js = (string) file_get_contents( REPUTECAST_DIR . '/blocks/signup-form/index.js' );
		$this->assertStringNotContainsString( 'import ', $js, 'plain JavaScript: no build step' );
		$this->assertStringContainsString( "registerBlockType( 'reputecast/signup-form'", $js );
		$this->assertStringContainsString( "path: '/reputecast/v1/forms'", $js );
		$this->assertFileExists( REPUTECAST_DIR . '/languages/reputecast.pot' );
	}

	public function test_the_block_renders_the_container(): void {
		$this->connect_site();
		Functions\when( 'wp_enqueue_script' )->justReturn( null );
		Functions\when( 'get_block_wrapper_attributes' )->alias(
			static function ( $extra ) {
				$out = array( 'class="wp-block-reputecast-signup-form"' );
				foreach ( $extra as $k => $v ) {
					$out[] = $k . '="' . htmlspecialchars( $v, ENT_QUOTES ) . '"';
				}
				return implode( ' ', $out );
			}
		);
		$attributes = array(
			'formId'   => 'frm_8c2e1a',
			'language' => 'es',
		);
		ob_start();
		require REPUTECAST_DIR . '/blocks/signup-form/render.php';
		$this->assertSame( '<div class="wp-block-reputecast-signup-form" data-reputecast-form="frm_8c2e1a" lang="es"></div>', ob_get_clean() );
		$attributes = array( 'formId' => '' );
		ob_start();
		require REPUTECAST_DIR . '/blocks/signup-form/render.php';
		$this->assertSame( '', ob_get_clean(), 'no form chosen yet: nothing on the page' );
	}

	/**
	 * A "Name: value" line of a plugin header or readme.
	 *
	 * @param string $text The file.
	 * @param string $name The header's name.
	 */
	private function header_value( string $text, string $name ): string {
		preg_match( '/^[\s*]*' . preg_quote( $name, '/' ) . ':\s*(.+)$/m', $text, $m );
		return trim( $m[1] ?? '' );
	}
}
