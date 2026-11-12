from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.whats_app_account import WhatsAppAccount
  from ..models.whats_app_list_consent import WhatsAppListConsent
  from ..models.whats_app_template import WhatsAppTemplate
  from ..models.whats_app_template_counts import WhatsAppTemplateCounts





T = TypeVar("T", bound="WhatsAppOverview")



@_attrs_define
class WhatsAppOverview:
    """ 
        Attributes:
            can_manage (bool):
            can_send (bool):
            consent (int):
            consent_lists (list[WhatsAppListConsent]):
            helpers (list[str]):
            quiet (bool):
            rates_as_of (datetime.datetime):
            same_window_as_sms (bool):
            template_counts (WhatsAppTemplateCounts):
            templates (list[WhatsAppTemplate]):
            test_numbers (list[str]):
            webhook_url (str):
            window (str):
            account (WhatsAppAccount | Unset):
     """

    can_manage: bool
    can_send: bool
    consent: int
    consent_lists: list[WhatsAppListConsent]
    helpers: list[str]
    quiet: bool
    rates_as_of: datetime.datetime
    same_window_as_sms: bool
    template_counts: WhatsAppTemplateCounts
    templates: list[WhatsAppTemplate]
    test_numbers: list[str]
    webhook_url: str
    window: str
    account: WhatsAppAccount | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.whats_app_account import WhatsAppAccount # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_template_counts import WhatsAppTemplateCounts # noqa: PLC0415
        can_manage = self.can_manage

        can_send = self.can_send

        consent = self.consent

        consent_lists = []
        for consent_lists_item_data in self.consent_lists:
            consent_lists_item = consent_lists_item_data.to_dict()
            consent_lists.append(consent_lists_item)



        helpers = self.helpers



        quiet = self.quiet

        rates_as_of = self.rates_as_of.isoformat()

        same_window_as_sms = self.same_window_as_sms

        template_counts = self.template_counts.to_dict()

        templates = []
        for templates_item_data in self.templates:
            templates_item = templates_item_data.to_dict()
            templates.append(templates_item)



        test_numbers = self.test_numbers



        webhook_url = self.webhook_url

        window = self.window

        account: dict[str, Any] | Unset = UNSET
        if not isinstance(self.account, Unset):
            account = self.account.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "canSend": can_send,
            "consent": consent,
            "consentLists": consent_lists,
            "helpers": helpers,
            "quiet": quiet,
            "ratesAsOf": rates_as_of,
            "sameWindowAsSms": same_window_as_sms,
            "templateCounts": template_counts,
            "templates": templates,
            "testNumbers": test_numbers,
            "webhookUrl": webhook_url,
            "window": window,
        })
        if account is not UNSET:
            field_dict["account"] = account

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.whats_app_account import WhatsAppAccount # noqa: PLC0415
        from ..models.whats_app_list_consent import WhatsAppListConsent # noqa: PLC0415
        from ..models.whats_app_template import WhatsAppTemplate # noqa: PLC0415
        from ..models.whats_app_template_counts import WhatsAppTemplateCounts # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        can_send = d.pop("canSend")

        consent = d.pop("consent")

        consent_lists = []
        _consent_lists = d.pop("consentLists")
        for consent_lists_item_data in (_consent_lists):
            consent_lists_item = WhatsAppListConsent.from_dict(consent_lists_item_data)



            consent_lists.append(consent_lists_item)


        helpers = cast(list[str], d.pop("helpers"))


        quiet = d.pop("quiet")

        rates_as_of = datetime.datetime.fromisoformat(d.pop("ratesAsOf"))




        same_window_as_sms = d.pop("sameWindowAsSms")

        template_counts = WhatsAppTemplateCounts.from_dict(d.pop("templateCounts"))




        templates = []
        _templates = d.pop("templates")
        for templates_item_data in (_templates):
            templates_item = WhatsAppTemplate.from_dict(templates_item_data)



            templates.append(templates_item)


        test_numbers = cast(list[str], d.pop("testNumbers"))


        webhook_url = d.pop("webhookUrl")

        window = d.pop("window")

        _account = d.pop("account", UNSET)
        account: WhatsAppAccount | Unset
        if isinstance(_account,  Unset):
            account = UNSET
        else:
            account = WhatsAppAccount.from_dict(_account)




        whats_app_overview = cls(
            can_manage=can_manage,
            can_send=can_send,
            consent=consent,
            consent_lists=consent_lists,
            helpers=helpers,
            quiet=quiet,
            rates_as_of=rates_as_of,
            same_window_as_sms=same_window_as_sms,
            template_counts=template_counts,
            templates=templates,
            test_numbers=test_numbers,
            webhook_url=webhook_url,
            window=window,
            account=account,
        )

        return whats_app_overview

