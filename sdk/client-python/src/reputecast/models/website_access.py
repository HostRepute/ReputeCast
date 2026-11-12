from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_access_campaign_visibility import WebsiteAccessCampaignVisibility
from ..models.website_access_default_visibility import WebsiteAccessDefaultVisibility
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.website_access_list import WebsiteAccessList
  from ..models.website_members_facts import WebsiteMembersFacts
  from ..models.website_never_free_post import WebsiteNeverFreePost
  from ..models.website_payments import WebsitePayments
  from ..models.website_paywall import WebsitePaywall
  from ..models.website_tier_option import WebsiteTierOption





T = TypeVar("T", bound="WebsiteAccess")



@_attrs_define
class WebsiteAccess:
    """ 
        Attributes:
            campaign_visibility (WebsiteAccessCampaignVisibility):
            default_visibility (WebsiteAccessDefaultVisibility):
            link_minutes (int):
            lists (list[WebsiteAccessList]):
            members (WebsiteMembersFacts):
            members_only_posts (int):
            meter_members (int):
            meter_members_on (bool):
            meter_visitors (int):
            meter_visitors_on (bool):
            never_free (list[WebsiteNeverFreePost]):
            paid_posts (int):
            payments (WebsitePayments):
            paywall (WebsitePaywall):
            sender_email (str):
            sender_name (str):
            session_days (int):
            signin_on (bool):
            signup_on (bool):
            since (datetime.datetime):
            system_provider (str):
            tiers (list[WebsiteTierOption]):
            updated_at (datetime.datetime):
            updated_by (str | Unset):
     """

    campaign_visibility: WebsiteAccessCampaignVisibility
    default_visibility: WebsiteAccessDefaultVisibility
    link_minutes: int
    lists: list[WebsiteAccessList]
    members: WebsiteMembersFacts
    members_only_posts: int
    meter_members: int
    meter_members_on: bool
    meter_visitors: int
    meter_visitors_on: bool
    never_free: list[WebsiteNeverFreePost]
    paid_posts: int
    payments: WebsitePayments
    paywall: WebsitePaywall
    sender_email: str
    sender_name: str
    session_days: int
    signin_on: bool
    signup_on: bool
    since: datetime.datetime
    system_provider: str
    tiers: list[WebsiteTierOption]
    updated_at: datetime.datetime
    updated_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_access_list import WebsiteAccessList # noqa: PLC0415
        from ..models.website_members_facts import WebsiteMembersFacts # noqa: PLC0415
        from ..models.website_never_free_post import WebsiteNeverFreePost # noqa: PLC0415
        from ..models.website_payments import WebsitePayments # noqa: PLC0415
        from ..models.website_paywall import WebsitePaywall # noqa: PLC0415
        from ..models.website_tier_option import WebsiteTierOption # noqa: PLC0415
        campaign_visibility = self.campaign_visibility.value

        default_visibility = self.default_visibility.value

        link_minutes = self.link_minutes

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        members = self.members.to_dict()

        members_only_posts = self.members_only_posts

        meter_members = self.meter_members

        meter_members_on = self.meter_members_on

        meter_visitors = self.meter_visitors

        meter_visitors_on = self.meter_visitors_on

        never_free = []
        for never_free_item_data in self.never_free:
            never_free_item = never_free_item_data.to_dict()
            never_free.append(never_free_item)



        paid_posts = self.paid_posts

        payments = self.payments.to_dict()

        paywall = self.paywall.to_dict()

        sender_email = self.sender_email

        sender_name = self.sender_name

        session_days = self.session_days

        signin_on = self.signin_on

        signup_on = self.signup_on

        since = self.since.isoformat()

        system_provider = self.system_provider

        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        updated_at = self.updated_at.isoformat()

        updated_by = self.updated_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaignVisibility": campaign_visibility,
            "defaultVisibility": default_visibility,
            "linkMinutes": link_minutes,
            "lists": lists,
            "members": members,
            "membersOnlyPosts": members_only_posts,
            "meterMembers": meter_members,
            "meterMembersOn": meter_members_on,
            "meterVisitors": meter_visitors,
            "meterVisitorsOn": meter_visitors_on,
            "neverFree": never_free,
            "paidPosts": paid_posts,
            "payments": payments,
            "paywall": paywall,
            "senderEmail": sender_email,
            "senderName": sender_name,
            "sessionDays": session_days,
            "signinOn": signin_on,
            "signupOn": signup_on,
            "since": since,
            "systemProvider": system_provider,
            "tiers": tiers,
            "updatedAt": updated_at,
        })
        if updated_by is not UNSET:
            field_dict["updatedBy"] = updated_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_access_list import WebsiteAccessList # noqa: PLC0415
        from ..models.website_members_facts import WebsiteMembersFacts # noqa: PLC0415
        from ..models.website_never_free_post import WebsiteNeverFreePost # noqa: PLC0415
        from ..models.website_payments import WebsitePayments # noqa: PLC0415
        from ..models.website_paywall import WebsitePaywall # noqa: PLC0415
        from ..models.website_tier_option import WebsiteTierOption # noqa: PLC0415
        d = dict(src_dict)
        campaign_visibility = WebsiteAccessCampaignVisibility(d.pop("campaignVisibility"))




        default_visibility = WebsiteAccessDefaultVisibility(d.pop("defaultVisibility"))




        link_minutes = d.pop("linkMinutes")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = WebsiteAccessList.from_dict(lists_item_data)



            lists.append(lists_item)


        members = WebsiteMembersFacts.from_dict(d.pop("members"))




        members_only_posts = d.pop("membersOnlyPosts")

        meter_members = d.pop("meterMembers")

        meter_members_on = d.pop("meterMembersOn")

        meter_visitors = d.pop("meterVisitors")

        meter_visitors_on = d.pop("meterVisitorsOn")

        never_free = []
        _never_free = d.pop("neverFree")
        for never_free_item_data in (_never_free):
            never_free_item = WebsiteNeverFreePost.from_dict(never_free_item_data)



            never_free.append(never_free_item)


        paid_posts = d.pop("paidPosts")

        payments = WebsitePayments.from_dict(d.pop("payments"))




        paywall = WebsitePaywall.from_dict(d.pop("paywall"))




        sender_email = d.pop("senderEmail")

        sender_name = d.pop("senderName")

        session_days = d.pop("sessionDays")

        signin_on = d.pop("signinOn")

        signup_on = d.pop("signupOn")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        system_provider = d.pop("systemProvider")

        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = WebsiteTierOption.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        updated_by = d.pop("updatedBy", UNSET)

        website_access = cls(
            campaign_visibility=campaign_visibility,
            default_visibility=default_visibility,
            link_minutes=link_minutes,
            lists=lists,
            members=members,
            members_only_posts=members_only_posts,
            meter_members=meter_members,
            meter_members_on=meter_members_on,
            meter_visitors=meter_visitors,
            meter_visitors_on=meter_visitors_on,
            never_free=never_free,
            paid_posts=paid_posts,
            payments=payments,
            paywall=paywall,
            sender_email=sender_email,
            sender_name=sender_name,
            session_days=session_days,
            signin_on=signin_on,
            signup_on=signup_on,
            since=since,
            system_provider=system_provider,
            tiers=tiers,
            updated_at=updated_at,
            updated_by=updated_by,
        )

        return website_access

