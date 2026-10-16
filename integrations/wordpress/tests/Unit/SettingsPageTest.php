<?php
/**
 * Settings › ReputeCast.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Functions;
use ReputeCast\Options;
use ReputeCast\Reporter;
use ReputeCast\Settings_Page;
use ReputeCast\Tests\TestCase;

final class SettingsPageTest extends TestCase {

	protected function setUp(): void {
		parent::setUp();
		$this->stub_site();
		Functions\when( 'current_user_can' )->justReturn( true );
		Functions\when( 'wp_nonce_field' )->alias(
			static function ( $action ) {
				echo '<input type="hidden" name="_wpnonce" value="nonce-' . $action . '">';
			}
		);
		Functions\when( 'submit_button' )->alias(
			static function ( $text ) {
				echo '<input type="submit" value="' . htmlspecialchars( $text ) . '">';
			}
		);
		Functions\when( 'human_time_diff' )->justReturn( '6 mins' );
		Functions\when( 'wp_date' )->justReturn( '16 Oct 2026 14:02' );
		Functions\when( 'get_edit_post_link' )->alias( static fn( $id ) => 'https://blog.acme.io/wp-admin/post.php?post=' . $id . '&action=edit' );
		Functions\when( 'wp_kses' )->alias( static fn( $html, $allowed ) => strip_tags( $html, array_keys( $allowed ) ) );
	}

	/** GET /integration with a WordPress site's key. */
	private function answer_me( string $integration = 'wordpress' ): void {
		$this->answer(
			'get-integration',
			200,
			array(
				'workspace'     => 'acme-newsletter',
				'workspaceName' => 'Acme Newsletter',
				'integration'   => $integration,
				'key'           => 'WordPress · blog.acme.io',
				'triggers'      => array(),
				'actions'       => array(),
				'permissions'   => array( 'subscribers:view', 'campaigns:edit' ),
			)
		);
	}

	private function answer_forms(): void {
		$this->answer( 'list-wordpress-forms', 200, array( 'items' => array() ) );
	}

	public function test_connect_checks_the_key_reports_in_and_fetches_the_forms(): void {
		$this->answer_me();
		$this->answer_report();
		$this->answer_forms();
		$notice = Settings_Page::do_connect( 'news.acme.io', ' ' . self::KEY . ' ' );
		$this->assertSame(
			array(
				'type'    => 'success',
				'message' => 'Connected to Acme Newsletter. ReputeCast lists this site as blog.acme.io.',
			),
			$notice
		);
		$this->assertSame( array( 'get-integration', 'report-wordpress-site', 'list-wordpress-forms' ), array_column( $this->calls(), 'operationId' ) );
		$this->assertTrue( Options::is_connected() );
		$this->assertSame( self::KEY, Options::key() );
		$c = Options::connection();
		$this->assertSame( self::INSTANCE, $c['instance_url'] );
		$this->assertSame( 'acme-newsletter', $c['workspace'] );
		$this->assertSame( Reporter::CRON_HOOK, $this->scheduled[0]['hook'] );
	}

	public function test_what_was_pasted_is_checked_first(): void {
		$this->assertSame( 'error', Settings_Page::do_connect( '', self::KEY )['type'] );
		$notice = Settings_Page::do_connect( 'news.acme.io', 'rc_live_5d1e…7b20' );
		$this->assertStringStartsWith( 'That isn’t a ReputeCast key.', $notice['message'] );
		$this->assertSame( array(), $this->requests );
		$this->assertFalse( Options::is_connected() );
	}

	public function test_another_integrations_key_is_refused(): void {
		$this->answer_me( 'zapier' );
		$notice = Settings_Page::do_connect( self::INSTANCE, self::KEY );
		$this->assertSame( 'error', $notice['type'] );
		$this->assertStringStartsWith( 'This key isn’t a WordPress site’s key.', $notice['message'] );
		$this->assertSame( '', Options::key(), 'nothing is kept' );
	}

	public function test_reputecasts_own_sentence_on_a_bad_key(): void {
		$this->answer( 'get-integration', 401, array( 'title' => 'Unauthorized', 'status' => 401, 'detail' => 'This key was revoked. Connect the site again to get a new key.', 'code' => 'key_revoked' ) );
		$notice = Settings_Page::do_connect( self::INSTANCE, self::KEY );
		$this->assertSame( 'This key was revoked. Connect the site again to get a new key.', $notice['message'] );
	}

	public function test_a_site_connected_to_another_workspace_puts_everything_back(): void {
		$this->answer_me();
		$this->answer( 'report-wordpress-site', 409, array( 'title' => 'Conflict', 'status' => 409, 'detail' => 'blog.acme.io is connected to Acme Engineering Blog. A site connects to one workspace: disconnect it there first.' ) );
		$notice = Settings_Page::do_connect( self::INSTANCE, self::KEY );
		$this->assertSame( 'error', $notice['type'] );
		$this->assertStringStartsWith( 'blog.acme.io is connected to Acme Engineering Blog.', $notice['message'] );
		$this->assertFalse( Options::is_connected() );
		$this->assertSame( '', Options::connection()['instance_url'] );
		$this->assertSame( array(), $this->scheduled );
	}

	public function test_connected_even_when_the_first_report_fails(): void {
		$this->answer_me();
		$this->answer_raw( 503, 'Service Unavailable' );
		$this->answer_forms();
		$notice = Settings_Page::do_connect( self::INSTANCE, self::KEY );
		$this->assertSame( 'warning', $notice['type'] );
		$this->assertSame( 'Connected to Acme Newsletter, but the first report didn’t arrive: ReputeCast answered HTTP 503. The site reports in again within 12 hours, or press Report in now.', $notice['message'] );
		$this->assertTrue( Options::is_connected() );
	}

	public function test_a_new_key_for_the_same_workspace_keeps_the_rest(): void {
		$this->connect_site(
			array(
				'forms'        => array( array( 'id' => 'frm_8c2e1a' ) ),
				'connected_at' => 1000,
			)
		);
		$this->answer_me();
		$this->answer_report();
		$this->answer( 'list-wordpress-forms', 401, array( 'title' => 'Unauthorized', 'status' => 401, 'detail' => 'Rate limited.' ) );
		$rotated = 'rc_live_' . str_repeat( '7', 64 );
		Settings_Page::do_connect( self::INSTANCE, $rotated );
		$this->assertSame( $rotated, Options::key() );
		$c = Options::connection();
		$this->assertSame( 1000, $c['connected_at'] );
		$this->assertSame( array( array( 'id' => 'frm_8c2e1a' ) ), $c['forms'] );
	}

	public function test_another_workspace_starts_afresh(): void {
		$this->connect_site(
			array(
				'workspace' => 'acme-engineering',
				'forms'     => array( array( 'id' => 'frm_8c2e1a' ) ),
				'posts'     => array( array( 'post' => 1 ) ),
			)
		);
		$this->answer_me();
		$this->answer_report();
		$this->answer_forms();
		Settings_Page::do_connect( self::INSTANCE, self::KEY );
		$c = Options::connection();
		$this->assertSame( 'acme-newsletter', $c['workspace'] );
		$this->assertSame( array(), $c['forms'] );
		$this->assertSame( array(), $c['posts'] );
	}

	public function test_disconnect_forgets_the_key_and_keeps_the_forms(): void {
		$this->connect_site( array( 'forms' => array( array( 'id' => 'frm_8c2e1a' ) ) ) );
		Reporter::schedule();
		$notice = Settings_Page::do_disconnect();
		$this->assertSame( 'success', $notice['type'] );
		$this->assertStringStartsWith( 'Disconnected from Acme Newsletter. Signup forms keep working.', $notice['message'] );
		$this->assertSame( '', Options::key() );
		$this->assertSame( array( array( 'id' => 'frm_8c2e1a' ) ), Options::connection()['forms'] );
		$this->assertSame( array(), $this->scheduled );
	}

	public function test_report_in_now(): void {
		$this->connect_site();
		$this->answer_report();
		$this->answer_forms();
		$this->assertSame(
			array(
				'type'    => 'success',
				'message' => 'Reported in. ReputeCast lists this site as blog.acme.io.',
			),
			Settings_Page::do_report()
		);
	}

	public function test_only_administrators_change_the_settings(): void {
		Functions\when( 'current_user_can' )->alias( static fn( $cap ) => 'edit_posts' === $cap );
		Functions\expect( 'wp_die' )->once()->andReturnUsing(
			static function () {
				throw new \RuntimeException( 'wp_die' );
			}
		);
		Functions\expect( 'check_admin_referer' )->never();
		$this->expectExceptionMessage( 'wp_die' );
		Settings_Page::connect();
	}

	public function test_every_action_checks_its_nonce(): void {
		$checked = array();
		Functions\when( 'check_admin_referer' )->alias(
			static function ( $action ) use ( &$checked ) {
				$checked[] = $action;
				throw new \RuntimeException( 'stop' );
			}
		);
		foreach ( array( 'connect', 'disconnect', 'report_now' ) as $handler ) {
			try {
				Settings_Page::$handler();
			} catch ( \RuntimeException $e ) {
				$this->assertSame( 'stop', $e->getMessage() );
			}
		}
		$this->assertSame( array( 'reputecast_connect', 'reputecast_disconnect', 'reputecast_report' ), $checked );
	}

	public function test_the_connect_form(): void {
		$this->transients['reputecast_notice_1'] = array(
			'type'         => 'error',
			'message'      => 'That isn’t a ReputeCast key. <script>',
			'instance_url' => 'https://news.acme.io/"><script>',
		);
		ob_start();
		Settings_Page::render();
		$html = (string) ob_get_clean();
		$this->assertStringContainsString( '<h1>ReputeCast</h1>', $html );
		$this->assertStringContainsString( '<div class="notice notice-error is-dismissible"><p>That isn’t a ReputeCast key. &lt;script&gt;</p></div>', $html );
		$this->assertStringContainsString( 'value="https://news.acme.io/&quot;&gt;&lt;script&gt;"', $html, 'the address typed comes back, escaped' );
		$this->assertStringContainsString( 'name="_wpnonce" value="nonce-reputecast_connect"', $html );
		$this->assertStringContainsString( 'type="password" id="reputecast-site-key" name="site_key"', $html );
		$this->assertStringContainsString( 'autocomplete="off"', $html );
		$this->assertArrayNotHasKey( 'reputecast_notice_1', $this->transients, 'a notice shows once' );
	}

	public function test_the_connected_page(): void {
		$this->connect_site(
			array(
				'workspace_name' => 'Acme <b>Newsletter</b>',
				'last_report_at' => time() - 360,
				'latest_plugin'  => '1.4.0',
				'forms'          => array(
					array(
						'id'          => 'frm_8c2e1a',
						'name'        => 'Newsletter signup',
						'status'      => 'live',
						'kind'        => 'embedded',
						'siteAllowed' => false,
						'script'      => '',
					),
				),
				'posts'          => array(
					array(
						'post'       => 812,
						'title'      => 'Scheduled rollouts are here',
						'at'         => time() - 60,
						'created'    => true,
						'reason'     => '',
						'campaignId' => '01a14383-b881-776e-a079-41690f27069d',
						'error'      => '',
					),
				),
			)
		);
		ob_start();
		Settings_Page::render();
		$html = (string) ob_get_clean();
		$this->assertStringContainsString( '<strong>Acme &lt;b&gt;Newsletter&lt;/b&gt;</strong>', $html );
		$this->assertStringContainsString( '<code>rc_live_5d1e…7b20</code>', $html );
		$this->assertStringNotContainsString( self::KEY, $html, 'the key is never shown again' );
		$this->assertStringContainsString( 'Update available: 1.4.0.', $html );
		$this->assertStringContainsString( 'blog.acme.io isn’t one of the form’s allowed domains', $html );
		$this->assertStringContainsString( '[reputecast_form id=&quot;frm_8c2e1a&quot;]', $html );
		$this->assertStringContainsString( 'href="https://news.acme.io/w/acme-newsletter/campaigns/01a14383-b881-776e-a079-41690f27069d/edit"', $html );
		$this->assertStringContainsString( 'https://blog.acme.io/wp-json/reputecast/v1/status', $html );
		foreach ( array( 'reputecast_report', 'reputecast_connect', 'reputecast_disconnect' ) as $action ) {
			$this->assertStringContainsString( 'value="nonce-' . $action . '"', $html );
		}
	}

	public function test_what_reputecast_did_with_a_post(): void {
		$c = Options::connection();
		$this->assertSame( 'No draft: the post isn’t in one of the rule’s categories.', Settings_Page::outcome( array( 'reason' => 'category' ), $c ) );
		$this->assertStringStartsWith( 'No draft: the site has no post-draft rule yet.', Settings_Page::outcome( array( 'reason' => 'no_rule' ), $c ) );
		$this->assertSame( 'Not sent: ReputeCast didn’t answer &lt;here&gt;', Settings_Page::outcome( array( 'error' => 'ReputeCast didn’t answer <here>' ), $c ) );
	}

	public function test_a_settings_link_on_the_plugins_screen(): void {
		$links = Settings_Page::action_links( array( 'deactivate' => '<a href="#">Deactivate</a>' ) );
		$this->assertSame( array( 'settings', 'deactivate' ), array_keys( $links ) );
		$this->assertStringContainsString( 'options-general.php?page=reputecast', $links['settings'] );
	}
}
