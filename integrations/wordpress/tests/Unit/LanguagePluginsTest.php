<?php
/**
 * WPML and Polylang (plugin 1.3.0): translations as language versions, and
 * the form's language. Each test runs in its own process: a language
 * plugin, once there, stays for the process.
 *
 * @package ReputeCast
 */

namespace ReputeCast\Tests\Unit;

use Brain\Monkey\Actions;
use Brain\Monkey\Filters;
use Brain\Monkey\Functions;
use PHPUnit\Framework\Attributes\PreserveGlobalState;
use PHPUnit\Framework\Attributes\RunTestsInSeparateProcesses;
use ReputeCast\Forms;
use ReputeCast\Languages;
use ReputeCast\Posts;
use ReputeCast\Reporter;
use ReputeCast\Tests\TestCase;

#[RunTestsInSeparateProcesses]
#[PreserveGlobalState( false )]
final class LanguagePluginsTest extends TestCase {

	/**
	 * A post and its translations: 812 in English, 813 in Spanish, 814 in
	 * French (a draft), 815 in German (password-protected).
	 *
	 * @return array<int, \WP_Post>
	 */
	private function posts(): array {
		$posts = array(
			812 => array( 'Scheduled rollouts are here', '2026-10-15 10:04:00', 'publish', '' ),
			813 => array( 'Los despliegues programados ya están aquí', '2026-10-15 12:30:00', 'publish', '' ),
			814 => array( 'Les déploiements programmés sont là', '2026-10-15 12:31:00', 'draft', '' ),
			815 => array( 'Geplante Rollouts sind da', '2026-10-15 12:32:00', 'publish', 'secret' ),
		);
		$out   = array();
		foreach ( $posts as $id => $p ) {
			$out[ $id ] = new \WP_Post(
				array(
					'ID'            => $id,
					'post_title'    => $p[0],
					'post_excerpt'  => 'Excerpt ' . $id,
					'post_date_gmt' => $p[1],
					'post_status'   => $p[2],
					'post_password' => $p[3],
				)
			);
		}
		Functions\when( 'get_post' )->alias( static fn( $id ) => $out[ (int) $id ] ?? null );
		Functions\when( 'get_post_type' )->justReturn( 'post' );
		Functions\when( 'get_the_title' )->alias( static fn( $p ) => $p->post_title );
		Functions\when( 'get_permalink' )->alias( static fn( $p ) => 'https://blog.acme.io/?p=' . $p->ID );
		Functions\when( 'get_the_post_thumbnail_url' )->justReturn( false );
		Functions\when( 'wp_get_post_categories' )->justReturn( array( 'announcements' ) );
		Functions\when( 'get_post_time' )->alias( static fn( $format, $gmt, $p ) => gmdate( $format, strtotime( $p->post_date_gmt . ' UTC' ) ) );
		return $out;
	}

	/** Polylang, with English the default and Spanish, French and German. */
	private function polylang( string $current = 'es' ): void {
		$langs = array(
			812 => 'en',
			813 => 'es',
			814 => 'fr',
			815 => 'de',
		);
		Functions\when( 'pll_current_language' )->justReturn( $current );
		Functions\when( 'pll_default_language' )->justReturn( 'en' );
		Functions\when( 'pll_languages_list' )->alias(
			static fn( $args ) => 'name' === $args['fields'] ? array( 'English', 'Español', 'Français', 'Deutsch', 'Português' ) : array( 'en', 'es', 'fr', 'de', 'pt-br' )
		);
		Functions\when( 'pll_get_post_language' )->alias( static fn( $id ) => $langs[ $id ] ?? false );
		Functions\when( 'pll_get_post_translations' )->justReturn( array_flip( $langs ) );
	}

	/** WPML, with English the default. */
	private function wpml( string $current = 'de' ): void {
		define( 'ICL_SITEPRESS_VERSION', '4.7.0' );
		$langs = array(
			812 => 'en',
			813 => 'es',
			814 => 'fr',
			815 => 'de',
		);
		Filters\expectApplied( 'wpml_current_language' )->andReturn( $current );
		Filters\expectApplied( 'wpml_default_language' )->andReturn( 'en' );
		Filters\expectApplied( 'wpml_active_languages' )->andReturn(
			array(
				'en'      => array(
					'code'        => 'en',
					'native_name' => 'English',
				),
				'zh-hans' => array(
					'code'        => 'zh-hans',
					'native_name' => '简体中文',
				),
			)
		);
		Filters\expectApplied( 'wpml_post_language_details' )->andReturnUsing( static fn( $none, $id ) => array( 'language_code' => $langs[ $id ] ) );
		Filters\expectApplied( 'wpml_element_trid' )->andReturn( 77 );
		$translations = array();
		foreach ( $langs as $id => $lang ) {
			$translations[ $lang ] = (object) array(
				'element_id'    => $id,
				'language_code' => $lang,
			);
		}
		Filters\expectApplied( 'wpml_get_element_translations' )->with( null, 77, 'post_post' )->andReturn( $translations );
	}

	public function test_language_codes_as_bcp_47_writes_them(): void {
		$this->assertSame( 'pt-BR', Languages::bcp47( 'pt-br' ) );
		$this->assertSame( 'en-US', Languages::bcp47( 'en_US' ) );
		$this->assertSame( 'zh-Hans', Languages::bcp47( 'zh-hans' ) );
		$this->assertSame( 'es', Languages::bcp47( 'ES' ) );
		$this->assertSame( 'sr-Latn-RS', Languages::bcp47( 'sr_latn_rs' ) );
		$this->assertSame( '', Languages::bcp47( '' ) );
		$this->assertSame( '', Languages::bcp47( 'english' ) );
		$this->assertSame( '', Languages::bcp47( 'en-"><' ) );
	}

	public function test_polylang_languages_and_translations(): void {
		$this->polylang();
		$this->posts();
		$this->assertSame( 'polylang', Languages::plugin() );
		$this->assertSame( 'es', Languages::current() );
		$this->assertSame( 'en', Languages::site_default() );
		$this->assertSame(
			array(
				'code' => 'pt-BR',
				'name' => 'Português',
			),
			Languages::site_languages()[4]
		);
		$this->assertSame( 'fr', Languages::of_post( 814 ) );
		$this->assertSame(
			array(
				'es' => 813,
				'fr' => 814,
				'de' => 815,
			),
			Languages::translations( 812 )
		);
	}

	public function test_wpml_languages_and_translations(): void {
		$this->wpml();
		$this->posts();
		$this->assertSame( 'wpml', Languages::plugin() );
		$this->assertSame( 'de', Languages::current() );
		$this->assertSame(
			array(
				array(
					'code' => 'en',
					'name' => 'English',
				),
				array(
					'code' => 'zh-Hans',
					'name' => '简体中文',
				),
			),
			Languages::site_languages()
		);
		$this->assertSame( 'es', Languages::of_post( 813 ) );
		$this->assertSame(
			array(
				'en' => 812,
				'fr' => 814,
				'de' => 815,
			),
			Languages::translations( 813 )
		);
	}

	public function test_the_post_in_the_default_language_leads_its_published_translations(): void {
		$this->polylang();
		$this->posts();
		// Published from the Spanish translation: the English post leads.
		list( $main, $translations ) = Posts::group( get_post( 813 ) );
		$this->assertSame( 812, $main->ID );
		// The French draft and the password-protected German post stay out.
		$this->assertSame( array( 813 ), array_map( static fn( $p ) => $p->ID, $translations ) );
	}

	public function test_without_the_default_language_the_first_published_leads(): void {
		$this->polylang();
		$posts                     = $this->posts();
		$posts[812]->post_status   = 'draft';
		$posts[815]->post_password = '';
		list( $main, $translations ) = Posts::group( $posts[815] );
		$this->assertSame( 813, $main->ID, 'Spanish was published first' );
		$this->assertSame( array( 815 ), array_map( static fn( $p ) => $p->ID, $translations ) );
	}

	public function test_the_draft_request_carries_the_translations(): void {
		$this->wpml( 'en' );
		$this->posts();
		$this->connect_site();
		$this->stub_site();
		$this->answer(
			'create-wordpress-draft',
			200,
			array(
				'created'    => true,
				'campaignId' => '01a14383-b881-776e-a079-41690f27069d',
				'name'       => 'Scheduled rollouts are here',
				'languages'  => array( 'es' ),
			)
		);
		$this->answer_report();
		Posts::send( 813 );

		$calls = $this->calls();
		$post  = $calls[0]['body']['post'];
		$this->assertSame( '812', $post['id'] );
		$this->assertSame( 'en', $post['language'] );
		$this->assertSame(
			array(
				array(
					'id'       => '813',
					'url'      => 'https://blog.acme.io/?p=813',
					'title'    => 'Los despliegues programados ya están aquí',
					'excerpt'  => 'Excerpt 813',
					'language' => 'es',
				),
			),
			$post['translations']
		);
		$this->assertSame( 'wpml', $calls[1]['body']['languagePlugin'] );
	}

	public function test_the_form_follows_the_pages_language(): void {
		$this->polylang( 'de' );
		$this->connect_site();
		Functions\when( 'wp_enqueue_script' )->justReturn( null );
		$this->assertSame(
			array(
				'data-reputecast-form' => 'frm_8c2e1a',
				'lang'                 => 'de',
			),
			Forms::prepare( 'frm_8c2e1a' )
		);
		$this->assertSame(
			array(
				'data-reputecast-form' => 'frm_8c2e1a',
				'lang'                 => 'es',
			),
			Forms::prepare( 'frm_8c2e1a', 'es' ),
			'the block setting wins'
		);
	}

	public function test_polylang_categories_in_the_default_language(): void {
		$this->polylang();
		Functions\expect( 'get_terms' )
			->once()
			->with( \Mockery::on( static fn( $args ) => 'en' === ( $args['lang'] ?? null ) && 'category' === $args['taxonomy'] && false === $args['hide_empty'] ) )
			->andReturn( array() );
		$this->assertSame( array(), Reporter::categories() );
	}

	public function test_wpml_categories_in_the_default_language(): void {
		$this->wpml( 'de' );
		$order = array();
		Actions\expectDone( 'wpml_switch_language' )->twice()->whenHappen(
			static function ( $lang ) use ( &$order ) {
				$order[] = $lang;
			}
		);
		Functions\when( 'get_terms' )->alias(
			static function () use ( &$order ) {
				$order[] = 'query';
				return array();
			}
		);
		Reporter::categories();
		$this->assertSame( array( 'en', 'query', 'de' ), $order, 'switched to the default language, then back' );
	}
}
