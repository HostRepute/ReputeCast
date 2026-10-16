/**
 * ReputeCast signup form: the block in the editor. Plain JavaScript on
 * WordPress's own packages, so the plugin has no build step. The page shows
 * the form itself (render.php); the editor shows which form it is, and warns
 * when the form won't take this site's signups.
 */
( function ( wp ) {
	'use strict';

	const el = wp.element.createElement;
	const { useState, useEffect } = wp.element;
	const { __, sprintf } = wp.i18n;
	const { useBlockProps, InspectorControls } = wp.blockEditor;
	const { Placeholder, SelectControl, Notice, Spinner, Button, PanelBody } = wp.components;

	const stroke = { fill: 'none', stroke: '#FFFFFF', strokeWidth: 2.4, strokeLinecap: 'round', strokeLinejoin: 'round' };
	const icon = el(
		'svg',
		{ xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 32 32', width: 24, height: 24, 'aria-hidden': true, focusable: false },
		el( 'rect', { width: 32, height: 32, rx: 8, fill: '#0F766E' } ),
		el( 'path', Object.assign( { d: 'M7 12h10.5M7 12v10.5A1.5 1.5 0 0 0 8.5 24h15a1.5 1.5 0 0 0 1.5-1.5V16.5' }, stroke ) ),
		el( 'path', Object.assign( { d: 'M7 12l9 7L27 8' }, stroke ) )
	);

	// One request for every block in the post; a refresh asks ReputeCast again.
	let pending = null;
	function load( refresh ) {
		if ( refresh || ! pending ) {
			pending = wp.apiFetch( { path: '/reputecast/v1/forms' + ( refresh ? '?refresh=1' : '' ) } ).catch( ( err ) => {
				pending = null;
				throw err;
			} );
		}
		return pending;
	}

	function useForms() {
		const [ state, setState ] = useState( { loading: true, data: null, error: '' } );
		const fetchList = ( refresh ) => {
			let live = true;
			setState( ( s ) => Object.assign( {}, s, { loading: true } ) );
			load( refresh ).then(
				( data ) => live && setState( { loading: false, data, error: data.error || '' } ),
				( err ) => live && setState( ( s ) => ( { loading: false, data: s.data, error: ( err && err.message ) || __( 'The list of forms didn’t load.', 'reputecast' ) } ) )
			);
			return () => {
				live = false;
			};
		};
		useEffect( () => fetchList( false ), [] );
		return [ state, () => fetchList( true ) ];
	}

	const statusName = ( status ) =>
		( { live: __( 'Live', 'reputecast' ), paused: __( 'Paused', 'reputecast' ), draft: __( 'Draft', 'reputecast' ) } )[ status ] || status;

	function formOptions( forms ) {
		return [ { label: __( 'Choose a form', 'reputecast' ), value: '' } ].concat(
			forms.map( ( f ) => ( {
				label: 'live' === f.status ? f.name : sprintf( /* translators: 1: a form's name, 2: its status. */ __( '%1$s (%2$s)', 'reputecast' ), f.name, statusName( f.status ) ),
				value: f.id,
			} ) )
		);
	}

	function notice( status, text ) {
		return el( Notice, { status, isDismissible: false, className: 'reputecast-block-notice' }, text );
	}

	/** Why a chosen form may not work on this site. */
	function warnings( data, form, formId, error ) {
		const out = [];
		if ( error ) {
			/* translators: %s: what went wrong. */
			out.push( notice( 'warning', sprintf( __( 'The list of forms didn’t update: %s', 'reputecast' ), error ) ) );
		}
		if ( ! data.connected ) {
			out.push( notice( 'info', __( 'This site isn’t connected to ReputeCast. The form still shows on the page.', 'reputecast' ) ) );
		}
		if ( ! form ) {
			if ( data.connected && ! error ) {
				out.push( notice( 'warning', sprintf( /* translators: %s: a form's id. */ __( 'The form %s isn’t in the workspace anymore. Choose another one.', 'reputecast' ), formId ) ) );
			}
			return out;
		}
		if ( ! form.siteAllowed ) {
			out.push(
				notice(
					'warning',
					/* translators: %s: the site's address. */
					sprintf( __( '%s isn’t one of this form’s allowed domains, so ReputeCast rejects its signups. Add it to the form’s allowed domains in ReputeCast › Forms.', 'reputecast' ), data.site )
				)
			);
		}
		if ( 'draft' === form.status ) {
			out.push( notice( 'warning', __( 'This form is a draft in ReputeCast, so it doesn’t show on the page yet. Publish it there.', 'reputecast' ) ) );
		} else if ( 'paused' === form.status ) {
			out.push( notice( 'info', __( 'This form is paused in ReputeCast: it shows on the page, but takes no signups until it’s resumed.', 'reputecast' ) ) );
		}
		return out;
	}

	function Edit( { attributes, setAttributes } ) {
		const blockProps = useBlockProps();
		const [ state, refresh ] = useForms();
		const { data, loading, error } = state;
		const forms = data ? data.forms : [];
		const form = forms.find( ( f ) => f.id === attributes.formId );
		const languages = data ? data.languages : [];
		const choose = ( formId ) => setAttributes( { formId } );

		const inspector = el(
			InspectorControls,
			null,
			el(
				PanelBody,
				{ title: __( 'Form', 'reputecast' ) },
				forms.length
					? el( SelectControl, {
							label: __( 'Form', 'reputecast' ),
							value: attributes.formId,
							options: formOptions( forms ),
							onChange: choose,
							__nextHasNoMarginBottom: true,
							__next40pxDefaultSize: true,
					  } )
					: null,
				el(
					'p',
					null,
					el(
						Button,
						{ variant: 'secondary', onClick: refresh, isBusy: loading, disabled: loading || ! ( data && data.connected ) },
						__( 'Refresh the list', 'reputecast' )
					)
				),
				attributes.formId
					? el( 'p', { className: 'reputecast-block-shortcode' }, __( 'Shortcode:', 'reputecast' ), ' ', el( 'code', null, '[reputecast_form id="' + attributes.formId + '"]' ) )
					: null
			),
			languages.length || attributes.language
				? el(
						PanelBody,
						{ title: __( 'Language', 'reputecast' ) },
						el( SelectControl, {
							label: __( 'Form language', 'reputecast' ),
							value: attributes.language,
							options: [ { label: __( 'Match the page', 'reputecast' ), value: '' } ].concat(
								languages.map( ( l ) => ( { label: l.name + ' (' + l.code + ')', value: l.code } ) ),
								attributes.language && ! languages.some( ( l ) => l.code === attributes.language ) ? [ { label: attributes.language, value: attributes.language } ] : []
							),
							help: __( 'The form shows its copy in this language when it has it in ReputeCast. Otherwise visitors see the form’s default language.', 'reputecast' ),
							onChange: ( language ) => setAttributes( { language } ),
							__nextHasNoMarginBottom: true,
							__next40pxDefaultSize: true,
						} )
				  )
				: null
		);

		const label = __( 'ReputeCast signup form', 'reputecast' );
		let body;
		if ( ! data && loading ) {
			body = el( Placeholder, { icon, label }, el( Spinner ) );
		} else if ( ! data ) {
			body = el( Placeholder, { icon, label }, notice( 'error', error ) );
		} else if ( ! attributes.formId ) {
			if ( forms.length ) {
				body = el(
					Placeholder,
					{ icon, label, instructions: __( 'Choose one of the workspace’s signup forms.', 'reputecast' ) },
					el( SelectControl, { label: __( 'Form', 'reputecast' ), hideLabelFromVision: true, value: '', options: formOptions( forms ), onChange: choose, __nextHasNoMarginBottom: true, __next40pxDefaultSize: true } )
				);
			} else if ( ! data.connected ) {
				body = el(
					Placeholder,
					{
						icon,
						label,
						instructions: data.canManage
							? __( 'Connect this site to ReputeCast to list its signup forms.', 'reputecast' )
							: __( 'Ask an administrator to connect this site to ReputeCast, in Settings › ReputeCast.', 'reputecast' ),
					},
					data.canManage ? el( Button, { variant: 'primary', href: data.settingsUrl }, __( 'Open Settings › ReputeCast', 'reputecast' ) ) : null
				);
			} else {
				body = el(
					Placeholder,
					{ icon, label, instructions: __( 'The workspace has no signup forms yet. Create one in ReputeCast › Forms, then refresh the list.', 'reputecast' ) },
					error ? notice( 'warning', error ) : null,
					el( Button, { variant: 'secondary', onClick: refresh, isBusy: loading, disabled: loading }, __( 'Refresh the list', 'reputecast' ) )
				);
			}
		} else {
			const popup = form && 'popup' === form.kind;
			body = el(
				'div',
				{ className: 'reputecast-block-preview' },
				el(
					'div',
					{ className: 'reputecast-block-preview__head' },
					icon,
					el( 'strong', null, form ? form.name : attributes.formId ),
					form ? el( 'span', { className: 'reputecast-block-preview__status is-' + form.status }, statusName( form.status ) ) : null
				),
				el(
					'p',
					{ className: 'reputecast-block-preview__text' },
					popup
						? __( 'This popup form opens over the page as set in ReputeCast.', 'reputecast' )
						: __( 'The form shows here on the published page, as it looks in ReputeCast.', 'reputecast' )
				),
				warnings( data, form, attributes.formId, error )
			);
		}
		return el( 'div', blockProps, inspector, body );
	}

	wp.blocks.registerBlockType( 'reputecast/signup-form', {
		icon,
		edit: Edit,
		save: () => null,
	} );
} )( window.wp );
