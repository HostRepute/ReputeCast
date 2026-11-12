from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_rules_input_keywords_by import SmsRulesInputKeywordsBy
from ..models.sms_rules_input_retention_days import SmsRulesInputRetentionDays
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_thresholds import SmsThresholds





T = TypeVar("T", bound="SmsRulesInput")



@_attrs_define
class SmsRulesInput:
    """ 
        Attributes:
            alert_pct (int):
            brand_prefix (str):
            budget (float):
            confirmation (str):
            confirmation_on (bool):
            help_keywords (list[str]):
            help_reply (str):
            keywords_by (SmsRulesInputKeywordsBy):
            opt_in_keywords (list[str]):
            opt_in_reply (str):
            opt_out_in_steps (bool):
            opt_out_keywords (list[str]):
            opt_out_reply (str):
            opt_out_text (str):
            push_window (bool):
            quiet_from (str):
            quiet_to (str):
            require_yes (bool):
            retention_days (SmsRulesInputRetentionDays):
            thresholds (SmsThresholds):
            whats_app_window (bool):
     """

    alert_pct: int
    brand_prefix: str
    budget: float
    confirmation: str
    confirmation_on: bool
    help_keywords: list[str]
    help_reply: str
    keywords_by: SmsRulesInputKeywordsBy
    opt_in_keywords: list[str]
    opt_in_reply: str
    opt_out_in_steps: bool
    opt_out_keywords: list[str]
    opt_out_reply: str
    opt_out_text: str
    push_window: bool
    quiet_from: str
    quiet_to: str
    require_yes: bool
    retention_days: SmsRulesInputRetentionDays
    thresholds: SmsThresholds
    whats_app_window: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        alert_pct = self.alert_pct

        brand_prefix = self.brand_prefix

        budget = self.budget

        confirmation = self.confirmation

        confirmation_on = self.confirmation_on

        help_keywords = self.help_keywords



        help_reply = self.help_reply

        keywords_by = self.keywords_by.value

        opt_in_keywords = self.opt_in_keywords



        opt_in_reply = self.opt_in_reply

        opt_out_in_steps = self.opt_out_in_steps

        opt_out_keywords = self.opt_out_keywords



        opt_out_reply = self.opt_out_reply

        opt_out_text = self.opt_out_text

        push_window = self.push_window

        quiet_from = self.quiet_from

        quiet_to = self.quiet_to

        require_yes = self.require_yes

        retention_days = self.retention_days.value

        thresholds = self.thresholds.to_dict()

        whats_app_window = self.whats_app_window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertPct": alert_pct,
            "brandPrefix": brand_prefix,
            "budget": budget,
            "confirmation": confirmation,
            "confirmationOn": confirmation_on,
            "helpKeywords": help_keywords,
            "helpReply": help_reply,
            "keywordsBy": keywords_by,
            "optInKeywords": opt_in_keywords,
            "optInReply": opt_in_reply,
            "optOutInSteps": opt_out_in_steps,
            "optOutKeywords": opt_out_keywords,
            "optOutReply": opt_out_reply,
            "optOutText": opt_out_text,
            "pushWindow": push_window,
            "quietFrom": quiet_from,
            "quietTo": quiet_to,
            "requireYes": require_yes,
            "retentionDays": retention_days,
            "thresholds": thresholds,
            "whatsAppWindow": whats_app_window,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        d = dict(src_dict)
        alert_pct = d.pop("alertPct")

        brand_prefix = d.pop("brandPrefix")

        budget = d.pop("budget")

        confirmation = d.pop("confirmation")

        confirmation_on = d.pop("confirmationOn")

        help_keywords = cast(list[str], d.pop("helpKeywords"))


        help_reply = d.pop("helpReply")

        keywords_by = SmsRulesInputKeywordsBy(d.pop("keywordsBy"))




        opt_in_keywords = cast(list[str], d.pop("optInKeywords"))


        opt_in_reply = d.pop("optInReply")

        opt_out_in_steps = d.pop("optOutInSteps")

        opt_out_keywords = cast(list[str], d.pop("optOutKeywords"))


        opt_out_reply = d.pop("optOutReply")

        opt_out_text = d.pop("optOutText")

        push_window = d.pop("pushWindow")

        quiet_from = d.pop("quietFrom")

        quiet_to = d.pop("quietTo")

        require_yes = d.pop("requireYes")

        retention_days = SmsRulesInputRetentionDays(d.pop("retentionDays"))




        thresholds = SmsThresholds.from_dict(d.pop("thresholds"))




        whats_app_window = d.pop("whatsAppWindow")

        sms_rules_input = cls(
            alert_pct=alert_pct,
            brand_prefix=brand_prefix,
            budget=budget,
            confirmation=confirmation,
            confirmation_on=confirmation_on,
            help_keywords=help_keywords,
            help_reply=help_reply,
            keywords_by=keywords_by,
            opt_in_keywords=opt_in_keywords,
            opt_in_reply=opt_in_reply,
            opt_out_in_steps=opt_out_in_steps,
            opt_out_keywords=opt_out_keywords,
            opt_out_reply=opt_out_reply,
            opt_out_text=opt_out_text,
            push_window=push_window,
            quiet_from=quiet_from,
            quiet_to=quiet_to,
            require_yes=require_yes,
            retention_days=retention_days,
            thresholds=thresholds,
            whats_app_window=whats_app_window,
        )

        return sms_rules_input

