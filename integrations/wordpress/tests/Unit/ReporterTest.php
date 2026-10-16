<?php
/**
 * Report in.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use ReputeCast\Options;
use ReputeCast\Reporter;
use ReputeCast\Tests\TestCase;

final class ReporterTest extends TestCase {

	public function test_the_report_is_what_the_api_takes(): void {
		$this->connect_site( array( 'latest_plugin' => '' ) );
		$this->stub_site(
			array(
				new \WP_Term(
					array(
						'term_id' => 3,
						'name'    => 'Announcements &amp; news',
						'slug'    => 'announcements',
					)
				),
				new \WP_Term(
					array(
						'term_id' => 4,
						'name'    => 'Événements',
						'slug'    => '%c3%a9v%c3%a9nements',
					)
				),
			)
		);
		$this->answer_report();
		$result = Reporter::report();
		$this->assertSame( 'blog.acme.io', $result['site'] );

		$call = $this->calls()[0];
		$this->assertSame( 'report-wordpress-site', $call['operationId'] );
		$this->assertSame(
			array(
				'siteUrl'          => 'https://blog.acme.io',
				'name'             => 'Acme Blog',
				'pluginVersion'    => REPUTECAST_VERSION,
				'wordpressVersion' => '6.9.2',
				'phpVersion'       => PHP_MAJOR_VERSION . '.' . PHP_MINOR_VERSION,
				'feedUrl'          => 'https://blog.acme.io/feed/',
				'languagePlugin'   => '',
				'categories'       => array(
					array(
						'id'   => '3',
						'name' => 'Announcements & news',
						'slug' => 'announcements',
					),
					array(
						'id'   => '4',
						'name' => 'Événements',
						'slug' => 'événements',
					),
				),
				'forms'            => array(),
			),
			$call['body']
		);

		$c = Options::connection();
		$this->assertSame( '1.3.0', $c['latest_plugin'] );
		$this->assertSame( '', $c['last_report_error'] );
		$this->assertGreaterThan( 0, $c['last_report_at'] );
	}

	public function test_a_failed_report_is_kept_for_the_settings_page(): void {
		$this->connect_site( array( 'last_report_at' => 1000 ) );
		$this->stub_site();
		$this->answer(
			'report-wordpress-site',
			409,
			array(
				'title'  => 'Conflict',
				'status' => 409,
				'detail' => 'blog.acme.io is connected to Acme Engineering Blog. A site connects to one workspace: disconnect it there first.',
			)
		);
		$err = Reporter::report();
		$this->assertSame( 409, $err->get_error_data()['status'] );
		$c = Options::connection();
		$this->assertSame( 'blog.acme.io is connected to Acme Engineering Blog. A site connects to one workspace: disconnect it there first.', $c['last_report_error'] );
		$this->assertSame( 1000, $c['last_report_at'], 'the last report that arrived' );
	}

	public function test_a_disconnected_site_doesnt_report(): void {
		$this->assertSame( 'reputecast_not_connected', Reporter::report()->get_error_code() );
		Reporter::run();
		$this->assertSame( array(), $this->requests );
	}

	public function test_every_12_hours(): void {
		Reporter::schedule();
		Reporter::schedule();
		$this->assertCount( 1, $this->scheduled );
		$this->assertSame( 'twicedaily', $this->scheduled[0]['recurrence'] );
		$this->assertSame( Reporter::CRON_HOOK, $this->scheduled[0]['hook'] );
		Reporter::unschedule();
		$this->assertSame( array(), $this->scheduled );
	}

	public function test_the_12_hour_job_also_refreshes_the_forms(): void {
		$this->connect_site();
		$this->stub_site();
		$this->answer_report();
		$this->answer( 'list-wordpress-forms', 200, array( 'items' => array() ) );
		Reporter::run();
		$this->assertSame( array( 'report-wordpress-site', 'list-wordpress-forms' ), array_column( $this->calls(), 'operationId' ) );
	}

	public function test_workspace_slugs(): void {
		$this->assertSame( 'acme-newsletter', Reporter::slug( 'acme-newsletter' ) );
		$this->assertSame( '', Reporter::slug( '../admin' ) );
		$this->assertSame( '', Reporter::slug( '' ) );
	}
}
