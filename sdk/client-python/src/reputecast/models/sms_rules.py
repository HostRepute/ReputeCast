from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_rules_keywords_by import SmsRulesKeywordsBy
from ..models.sms_rules_other import SmsRulesOther
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_member_option import SmsMemberOption
  from ..models.sms_reply_view import SmsReplyView
  from ..models.sms_route_gap import SmsRouteGap
  from ..models.sms_route_view import SmsRouteView
  from ..models.sms_sender_option import SmsSenderOption
  from ..models.sms_short_domain import SmsShortDomain
  from ..models.sms_test_number import SmsTestNumber
  from ..models.sms_thresholds import SmsThresholds
  from ..models.sms_zone_now import SmsZoneNow





T = TypeVar("T", bound="SmsRules")



@_attrs_define
class SmsRules:
    """ 
        Attributes:
            alert_pct (int):
            assumed (int):
            brand_prefix (str):
            budget (float):
            can_manage (bool):
            confirmation (SmsReplyView):
            confirmation_on (bool):
            gaps (list[SmsRouteGap]):
            help_keywords (list[str]):
            help_reply (SmsReplyView):
            helpers (list[str]):
            keywords_by (SmsRulesKeywordsBy):
            members (list[SmsMemberOption]):
            month (datetime.datetime):
            opt_in_keywords (list[str]):
            opt_in_reply (SmsReplyView):
            opt_out_in_steps (bool):
            opt_out_keywords (list[str]):
            opt_out_reply (SmsReplyView):
            opt_out_text (str):
            other (SmsRulesOther):
            other_people (int):
            provider_answers (list[str]):
            push_window (bool):
            quiet_from (str):
            quiet_to (str):
            require_yes (bool):
            retention_days (int):
            routes (list[SmsRouteView]):
            senders (list[SmsSenderOption]):
            short_domain (SmsShortDomain):
            spent (float):
            test_numbers (list[SmsTestNumber]):
            tests_cost (float):
            tests_sent (int):
            thresholds (SmsThresholds):
            timezone (str):
            whats_app_window (bool):
            zones (list[SmsZoneNow]):
            budget_set_at (datetime.datetime | Unset):
            budget_set_by (str | Unset):
            other_route_id (str | Unset):
            updated_at (datetime.datetime | Unset):
            updated_by (str | Unset):
     """

    alert_pct: int
    assumed: int
    brand_prefix: str
    budget: float
    can_manage: bool
    confirmation: SmsReplyView
    confirmation_on: bool
    gaps: list[SmsRouteGap]
    help_keywords: list[str]
    help_reply: SmsReplyView
    helpers: list[str]
    keywords_by: SmsRulesKeywordsBy
    members: list[SmsMemberOption]
    month: datetime.datetime
    opt_in_keywords: list[str]
    opt_in_reply: SmsReplyView
    opt_out_in_steps: bool
    opt_out_keywords: list[str]
    opt_out_reply: SmsReplyView
    opt_out_text: str
    other: SmsRulesOther
    other_people: int
    provider_answers: list[str]
    push_window: bool
    quiet_from: str
    quiet_to: str
    require_yes: bool
    retention_days: int
    routes: list[SmsRouteView]
    senders: list[SmsSenderOption]
    short_domain: SmsShortDomain
    spent: float
    test_numbers: list[SmsTestNumber]
    tests_cost: float
    tests_sent: int
    thresholds: SmsThresholds
    timezone: str
    whats_app_window: bool
    zones: list[SmsZoneNow]
    budget_set_at: datetime.datetime | Unset = UNSET
    budget_set_by: str | Unset = UNSET
    other_route_id: str | Unset = UNSET
    updated_at: datetime.datetime | Unset = UNSET
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_member_option import SmsMemberOption # noqa: PLC0415
        from ..models.sms_reply_view import SmsReplyView # noqa: PLC0415
        from ..models.sms_route_gap import SmsRouteGap # noqa: PLC0415
        from ..models.sms_route_view import SmsRouteView # noqa: PLC0415
        from ..models.sms_sender_option import SmsSenderOption # noqa: PLC0415
        from ..models.sms_short_domain import SmsShortDomain # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        from ..models.sms_zone_now import SmsZoneNow # noqa: PLC0415
        alert_pct = self.alert_pct

        assumed = self.assumed

        brand_prefix = self.brand_prefix

        budget = self.budget

        can_manage = self.can_manage

        confirmation = self.confirmation.to_dict()

        confirmation_on = self.confirmation_on

        gaps = []
        for gaps_item_data in self.gaps:
            gaps_item = gaps_item_data.to_dict()
            gaps.append(gaps_item)



        help_keywords = self.help_keywords



        help_reply = self.help_reply.to_dict()

        helpers = self.helpers



        keywords_by = self.keywords_by.value

        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        month = self.month.isoformat()

        opt_in_keywords = self.opt_in_keywords



        opt_in_reply = self.opt_in_reply.to_dict()

        opt_out_in_steps = self.opt_out_in_steps

        opt_out_keywords = self.opt_out_keywords



        opt_out_reply = self.opt_out_reply.to_dict()

        opt_out_text = self.opt_out_text

        other = self.other.value

        other_people = self.other_people

        provider_answers = self.provider_answers



        push_window = self.push_window

        quiet_from = self.quiet_from

        quiet_to = self.quiet_to

        require_yes = self.require_yes

        retention_days = self.retention_days

        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        short_domain = self.short_domain.to_dict()

        spent = self.spent

        test_numbers = []
        for test_numbers_item_data in self.test_numbers:
            test_numbers_item = test_numbers_item_data.to_dict()
            test_numbers.append(test_numbers_item)



        tests_cost = self.tests_cost

        tests_sent = self.tests_sent

        thresholds = self.thresholds.to_dict()

        timezone = self.timezone

        whats_app_window = self.whats_app_window

        zones = []
        for zones_item_data in self.zones:
            zones_item = zones_item_data.to_dict()
            zones.append(zones_item)



        budget_set_at: str | Unset = UNSET
        if not isinstance(self.budget_set_at, Unset):
            budget_set_at = self.budget_set_at.isoformat()

        budget_set_by = self.budget_set_by

        other_route_id = self.other_route_id

        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alertPct": alert_pct,
            "assumed": assumed,
            "brandPrefix": brand_prefix,
            "budget": budget,
            "canManage": can_manage,
            "confirmation": confirmation,
            "confirmationOn": confirmation_on,
            "gaps": gaps,
            "helpKeywords": help_keywords,
            "helpReply": help_reply,
            "helpers": helpers,
            "keywordsBy": keywords_by,
            "members": members,
            "month": month,
            "optInKeywords": opt_in_keywords,
            "optInReply": opt_in_reply,
            "optOutInSteps": opt_out_in_steps,
            "optOutKeywords": opt_out_keywords,
            "optOutReply": opt_out_reply,
            "optOutText": opt_out_text,
            "other": other,
            "otherPeople": other_people,
            "providerAnswers": provider_answers,
            "pushWindow": push_window,
            "quietFrom": quiet_from,
            "quietTo": quiet_to,
            "requireYes": require_yes,
            "retentionDays": retention_days,
            "routes": routes,
            "senders": senders,
            "shortDomain": short_domain,
            "spent": spent,
            "testNumbers": test_numbers,
            "testsCost": tests_cost,
            "testsSent": tests_sent,
            "thresholds": thresholds,
            "timezone": timezone,
            "whatsAppWindow": whats_app_window,
            "zones": zones,
        })
        if budget_set_at is not UNSET:
            field_dict["budgetSetAt"] = budget_set_at
        if budget_set_by is not UNSET:
            field_dict["budgetSetBy"] = budget_set_by
        if other_route_id is not UNSET:
            field_dict["otherRouteId"] = other_route_id
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_member_option import SmsMemberOption # noqa: PLC0415
        from ..models.sms_reply_view import SmsReplyView # noqa: PLC0415
        from ..models.sms_route_gap import SmsRouteGap # noqa: PLC0415
        from ..models.sms_route_view import SmsRouteView # noqa: PLC0415
        from ..models.sms_sender_option import SmsSenderOption # noqa: PLC0415
        from ..models.sms_short_domain import SmsShortDomain # noqa: PLC0415
        from ..models.sms_test_number import SmsTestNumber # noqa: PLC0415
        from ..models.sms_thresholds import SmsThresholds # noqa: PLC0415
        from ..models.sms_zone_now import SmsZoneNow # noqa: PLC0415
        d = dict(src_dict)
        alert_pct = d.pop("alertPct")

        assumed = d.pop("assumed")

        brand_prefix = d.pop("brandPrefix")

        budget = d.pop("budget")

        can_manage = d.pop("canManage")

        confirmation = SmsReplyView.from_dict(d.pop("confirmation"))




        confirmation_on = d.pop("confirmationOn")

        gaps = []
        _gaps = d.pop("gaps")
        for gaps_item_data in (_gaps):
            gaps_item = SmsRouteGap.from_dict(gaps_item_data)



            gaps.append(gaps_item)


        help_keywords = cast(list[str], d.pop("helpKeywords"))


        help_reply = SmsReplyView.from_dict(d.pop("helpReply"))




        helpers = cast(list[str], d.pop("helpers"))


        keywords_by = SmsRulesKeywordsBy(d.pop("keywordsBy"))




        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = SmsMemberOption.from_dict(members_item_data)



            members.append(members_item)


        month = datetime.datetime.fromisoformat(d.pop("month"))




        opt_in_keywords = cast(list[str], d.pop("optInKeywords"))


        opt_in_reply = SmsReplyView.from_dict(d.pop("optInReply"))




        opt_out_in_steps = d.pop("optOutInSteps")

        opt_out_keywords = cast(list[str], d.pop("optOutKeywords"))


        opt_out_reply = SmsReplyView.from_dict(d.pop("optOutReply"))




        opt_out_text = d.pop("optOutText")

        other = SmsRulesOther(d.pop("other"))




        other_people = d.pop("otherPeople")

        provider_answers = cast(list[str], d.pop("providerAnswers"))


        push_window = d.pop("pushWindow")

        quiet_from = d.pop("quietFrom")

        quiet_to = d.pop("quietTo")

        require_yes = d.pop("requireYes")

        retention_days = d.pop("retentionDays")

        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = SmsRouteView.from_dict(routes_item_data)



            routes.append(routes_item)


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsSenderOption.from_dict(senders_item_data)



            senders.append(senders_item)


        short_domain = SmsShortDomain.from_dict(d.pop("shortDomain"))




        spent = d.pop("spent")

        test_numbers = []
        _test_numbers = d.pop("testNumbers")
        for test_numbers_item_data in (_test_numbers):
            test_numbers_item = SmsTestNumber.from_dict(test_numbers_item_data)



            test_numbers.append(test_numbers_item)


        tests_cost = d.pop("testsCost")

        tests_sent = d.pop("testsSent")

        thresholds = SmsThresholds.from_dict(d.pop("thresholds"))




        timezone = d.pop("timezone")

        whats_app_window = d.pop("whatsAppWindow")

        zones = []
        _zones = d.pop("zones")
        for zones_item_data in (_zones):
            zones_item = SmsZoneNow.from_dict(zones_item_data)



            zones.append(zones_item)


        _budget_set_at = d.pop("budgetSetAt", UNSET)
        budget_set_at: datetime.datetime | Unset
        if isinstance(_budget_set_at,  Unset):
            budget_set_at = UNSET
        else:
            budget_set_at = datetime.datetime.fromisoformat(_budget_set_at)




        budget_set_by = d.pop("budgetSetBy", UNSET)

        other_route_id = d.pop("otherRouteId", UNSET)

        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        updated_by = d.pop("updatedBy", UNSET)

        sms_rules = cls(
            alert_pct=alert_pct,
            assumed=assumed,
            brand_prefix=brand_prefix,
            budget=budget,
            can_manage=can_manage,
            confirmation=confirmation,
            confirmation_on=confirmation_on,
            gaps=gaps,
            help_keywords=help_keywords,
            help_reply=help_reply,
            helpers=helpers,
            keywords_by=keywords_by,
            members=members,
            month=month,
            opt_in_keywords=opt_in_keywords,
            opt_in_reply=opt_in_reply,
            opt_out_in_steps=opt_out_in_steps,
            opt_out_keywords=opt_out_keywords,
            opt_out_reply=opt_out_reply,
            opt_out_text=opt_out_text,
            other=other,
            other_people=other_people,
            provider_answers=provider_answers,
            push_window=push_window,
            quiet_from=quiet_from,
            quiet_to=quiet_to,
            require_yes=require_yes,
            retention_days=retention_days,
            routes=routes,
            senders=senders,
            short_domain=short_domain,
            spent=spent,
            test_numbers=test_numbers,
            tests_cost=tests_cost,
            tests_sent=tests_sent,
            thresholds=thresholds,
            timezone=timezone,
            whats_app_window=whats_app_window,
            zones=zones,
            budget_set_at=budget_set_at,
            budget_set_by=budget_set_by,
            other_route_id=other_route_id,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return sms_rules

