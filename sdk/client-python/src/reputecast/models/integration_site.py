from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_site_status import IntegrationSiteStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.integration_item_state import IntegrationItemState
  from ..models.integration_key_view import IntegrationKeyView
  from ..models.integration_reason import IntegrationReason
  from ..models.integration_site_draft import IntegrationSiteDraft
  from ..models.integration_site_form import IntegrationSiteForm
  from ..models.integration_site_rule import IntegrationSiteRule
  from ..models.integration_site_woo import IntegrationSiteWoo
  from ..models.word_press_category import WordPressCategory
  from ..models.word_press_check import WordPressCheck





T = TypeVar("T", bound="IntegrationSite")



@_attrs_define
class IntegrationSite:
    """ 
        Attributes:
            categories (list[WordPressCategory]):
            connected_at (datetime.datetime):
            drafts (list[IntegrationSiteDraft]):
            drafts_this_week (int):
            forms (list[IntegrationSiteForm]):
            host (str):
            id (str): The connection's id
            items (list[IntegrationItemState]):
            reasons (list[IntegrationReason]):
            rule (IntegrationSiteRule):
            signups_rejected (bool):
            status (IntegrationSiteStatus):
            update_available (bool):
            connected_by (str | Unset):
            feed_url (str | Unset):
            first_report_at (datetime.datetime | Unset):
            key (IntegrationKeyView | Unset):
            language_plugin (str | Unset):
            last_check (WordPressCheck | Unset):
            last_report_at (datetime.datetime | Unset):
            name (str | Unset):
            old_key (IntegrationKeyView | Unset):
            php_version (str | Unset):
            plugin_version (str | Unset):
            site_url (str | Unset):
            woocommerce (IntegrationSiteWoo | Unset):
            wordpress_version (str | Unset):
     """

    categories: list[WordPressCategory]
    connected_at: datetime.datetime
    drafts: list[IntegrationSiteDraft]
    drafts_this_week: int
    forms: list[IntegrationSiteForm]
    host: str
    id: str
    items: list[IntegrationItemState]
    reasons: list[IntegrationReason]
    rule: IntegrationSiteRule
    signups_rejected: bool
    status: IntegrationSiteStatus
    update_available: bool
    connected_by: str | Unset = UNSET
    feed_url: str | Unset = UNSET
    first_report_at: datetime.datetime | Unset = UNSET
    key: IntegrationKeyView | Unset = UNSET
    language_plugin: str | Unset = UNSET
    last_check: WordPressCheck | Unset = UNSET
    last_report_at: datetime.datetime | Unset = UNSET
    name: str | Unset = UNSET
    old_key: IntegrationKeyView | Unset = UNSET
    php_version: str | Unset = UNSET
    plugin_version: str | Unset = UNSET
    site_url: str | Unset = UNSET
    woocommerce: IntegrationSiteWoo | Unset = UNSET
    wordpress_version: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.integration_item_state import IntegrationItemState # noqa: PLC0415
        from ..models.integration_key_view import IntegrationKeyView # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_site_draft import IntegrationSiteDraft # noqa: PLC0415
        from ..models.integration_site_form import IntegrationSiteForm # noqa: PLC0415
        from ..models.integration_site_rule import IntegrationSiteRule # noqa: PLC0415
        from ..models.integration_site_woo import IntegrationSiteWoo # noqa: PLC0415
        from ..models.word_press_category import WordPressCategory # noqa: PLC0415
        from ..models.word_press_check import WordPressCheck # noqa: PLC0415
        categories = []
        for categories_item_data in self.categories:
            categories_item = categories_item_data.to_dict()
            categories.append(categories_item)



        connected_at = self.connected_at.isoformat()

        drafts = []
        for drafts_item_data in self.drafts:
            drafts_item = drafts_item_data.to_dict()
            drafts.append(drafts_item)



        drafts_this_week = self.drafts_this_week

        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        host = self.host

        id = self.id

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        reasons = []
        for reasons_item_data in self.reasons:
            reasons_item = reasons_item_data.to_dict()
            reasons.append(reasons_item)



        rule = self.rule.to_dict()

        signups_rejected = self.signups_rejected

        status = self.status.value

        update_available = self.update_available

        connected_by = self.connected_by

        feed_url = self.feed_url

        first_report_at: str | Unset = UNSET
        if not isinstance(self.first_report_at, Unset):
            first_report_at = self.first_report_at.isoformat()

        key: dict[str, Any] | Unset = UNSET
        if not isinstance(self.key, Unset):
            key = self.key.to_dict()

        language_plugin = self.language_plugin

        last_check: dict[str, Any] | Unset = UNSET
        if not isinstance(self.last_check, Unset):
            last_check = self.last_check.to_dict()

        last_report_at: str | Unset = UNSET
        if not isinstance(self.last_report_at, Unset):
            last_report_at = self.last_report_at.isoformat()

        name = self.name

        old_key: dict[str, Any] | Unset = UNSET
        if not isinstance(self.old_key, Unset):
            old_key = self.old_key.to_dict()

        php_version = self.php_version

        plugin_version = self.plugin_version

        site_url = self.site_url

        woocommerce: dict[str, Any] | Unset = UNSET
        if not isinstance(self.woocommerce, Unset):
            woocommerce = self.woocommerce.to_dict()

        wordpress_version = self.wordpress_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "categories": categories,
            "connectedAt": connected_at,
            "drafts": drafts,
            "draftsThisWeek": drafts_this_week,
            "forms": forms,
            "host": host,
            "id": id,
            "items": items,
            "reasons": reasons,
            "rule": rule,
            "signupsRejected": signups_rejected,
            "status": status,
            "updateAvailable": update_available,
        })
        if connected_by is not UNSET:
            field_dict["connectedBy"] = connected_by
        if feed_url is not UNSET:
            field_dict["feedUrl"] = feed_url
        if first_report_at is not UNSET:
            field_dict["firstReportAt"] = first_report_at
        if key is not UNSET:
            field_dict["key"] = key
        if language_plugin is not UNSET:
            field_dict["languagePlugin"] = language_plugin
        if last_check is not UNSET:
            field_dict["lastCheck"] = last_check
        if last_report_at is not UNSET:
            field_dict["lastReportAt"] = last_report_at
        if name is not UNSET:
            field_dict["name"] = name
        if old_key is not UNSET:
            field_dict["oldKey"] = old_key
        if php_version is not UNSET:
            field_dict["phpVersion"] = php_version
        if plugin_version is not UNSET:
            field_dict["pluginVersion"] = plugin_version
        if site_url is not UNSET:
            field_dict["siteUrl"] = site_url
        if woocommerce is not UNSET:
            field_dict["woocommerce"] = woocommerce
        if wordpress_version is not UNSET:
            field_dict["wordpressVersion"] = wordpress_version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.integration_item_state import IntegrationItemState # noqa: PLC0415
        from ..models.integration_key_view import IntegrationKeyView # noqa: PLC0415
        from ..models.integration_reason import IntegrationReason # noqa: PLC0415
        from ..models.integration_site_draft import IntegrationSiteDraft # noqa: PLC0415
        from ..models.integration_site_form import IntegrationSiteForm # noqa: PLC0415
        from ..models.integration_site_rule import IntegrationSiteRule # noqa: PLC0415
        from ..models.integration_site_woo import IntegrationSiteWoo # noqa: PLC0415
        from ..models.word_press_category import WordPressCategory # noqa: PLC0415
        from ..models.word_press_check import WordPressCheck # noqa: PLC0415
        d = dict(src_dict)
        categories = []
        _categories = d.pop("categories")
        for categories_item_data in (_categories):
            categories_item = WordPressCategory.from_dict(categories_item_data)



            categories.append(categories_item)


        connected_at = datetime.datetime.fromisoformat(d.pop("connectedAt"))




        drafts = []
        _drafts = d.pop("drafts")
        for drafts_item_data in (_drafts):
            drafts_item = IntegrationSiteDraft.from_dict(drafts_item_data)



            drafts.append(drafts_item)


        drafts_this_week = d.pop("draftsThisWeek")

        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = IntegrationSiteForm.from_dict(forms_item_data)



            forms.append(forms_item)


        host = d.pop("host")

        id = d.pop("id")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = IntegrationItemState.from_dict(items_item_data)



            items.append(items_item)


        reasons = []
        _reasons = d.pop("reasons")
        for reasons_item_data in (_reasons):
            reasons_item = IntegrationReason.from_dict(reasons_item_data)



            reasons.append(reasons_item)


        rule = IntegrationSiteRule.from_dict(d.pop("rule"))




        signups_rejected = d.pop("signupsRejected")

        status = IntegrationSiteStatus(d.pop("status"))




        update_available = d.pop("updateAvailable")

        connected_by = d.pop("connectedBy", UNSET)

        feed_url = d.pop("feedUrl", UNSET)

        _first_report_at = d.pop("firstReportAt", UNSET)
        first_report_at: datetime.datetime | Unset
        if isinstance(_first_report_at,  Unset):
            first_report_at = UNSET
        else:
            first_report_at = datetime.datetime.fromisoformat(_first_report_at)




        _key = d.pop("key", UNSET)
        key: IntegrationKeyView | Unset
        if isinstance(_key,  Unset):
            key = UNSET
        else:
            key = IntegrationKeyView.from_dict(_key)




        language_plugin = d.pop("languagePlugin", UNSET)

        _last_check = d.pop("lastCheck", UNSET)
        last_check: WordPressCheck | Unset
        if isinstance(_last_check,  Unset):
            last_check = UNSET
        else:
            last_check = WordPressCheck.from_dict(_last_check)




        _last_report_at = d.pop("lastReportAt", UNSET)
        last_report_at: datetime.datetime | Unset
        if isinstance(_last_report_at,  Unset):
            last_report_at = UNSET
        else:
            last_report_at = datetime.datetime.fromisoformat(_last_report_at)




        name = d.pop("name", UNSET)

        _old_key = d.pop("oldKey", UNSET)
        old_key: IntegrationKeyView | Unset
        if isinstance(_old_key,  Unset):
            old_key = UNSET
        else:
            old_key = IntegrationKeyView.from_dict(_old_key)




        php_version = d.pop("phpVersion", UNSET)

        plugin_version = d.pop("pluginVersion", UNSET)

        site_url = d.pop("siteUrl", UNSET)

        _woocommerce = d.pop("woocommerce", UNSET)
        woocommerce: IntegrationSiteWoo | Unset
        if isinstance(_woocommerce,  Unset):
            woocommerce = UNSET
        else:
            woocommerce = IntegrationSiteWoo.from_dict(_woocommerce)




        wordpress_version = d.pop("wordpressVersion", UNSET)

        integration_site = cls(
            categories=categories,
            connected_at=connected_at,
            drafts=drafts,
            drafts_this_week=drafts_this_week,
            forms=forms,
            host=host,
            id=id,
            items=items,
            reasons=reasons,
            rule=rule,
            signups_rejected=signups_rejected,
            status=status,
            update_available=update_available,
            connected_by=connected_by,
            feed_url=feed_url,
            first_report_at=first_report_at,
            key=key,
            language_plugin=language_plugin,
            last_check=last_check,
            last_report_at=last_report_at,
            name=name,
            old_key=old_key,
            php_version=php_version,
            plugin_version=plugin_version,
            site_url=site_url,
            woocommerce=woocommerce,
            wordpress_version=wordpress_version,
        )

        return integration_site

