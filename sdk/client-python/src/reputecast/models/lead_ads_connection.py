from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.lead_ads_connection_platform import LeadAdsConnectionPlatform
from ..models.lead_ads_connection_problem import LeadAdsConnectionProblem
from ..models.lead_ads_connection_status import LeadAdsConnectionStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.lead_ads_page import LeadAdsPage
  from ..models.lead_ads_receiving import LeadAdsReceiving
  from ..models.lead_ads_webhook import LeadAdsWebhook





T = TypeVar("T", bound="LeadAdsConnection")



@_attrs_define
class LeadAdsConnection:
    """ 
        Attributes:
            app_id (str):  Example: 481516234200.
            can_manage (bool):
            created_at (datetime.datetime):
            created_by (str):
            helpers (list[str]):
            id (str):
            leads30d (int): Leads in the last 30 days, Meta's test leads left out
            name (str):  Example: Acme Marketing.
            pages (list[LeadAdsPage]): The chosen Pages and their lead forms
            platform (LeadAdsConnectionPlatform):
            receiving (LeadAdsReceiving):
            status (LeadAdsConnectionStatus):
            checked_at (datetime.datetime | Unset): When the token was last checked
            disconnected_at (datetime.datetime | Unset):
            problem (LeadAdsConnectionProblem | Unset):
            rate_limited_until (datetime.datetime | Unset):
            webhook (LeadAdsWebhook | Unset):
     """

    app_id: str
    can_manage: bool
    created_at: datetime.datetime
    created_by: str
    helpers: list[str]
    id: str
    leads30d: int
    name: str
    pages: list[LeadAdsPage]
    platform: LeadAdsConnectionPlatform
    receiving: LeadAdsReceiving
    status: LeadAdsConnectionStatus
    checked_at: datetime.datetime | Unset = UNSET
    disconnected_at: datetime.datetime | Unset = UNSET
    problem: LeadAdsConnectionProblem | Unset = UNSET
    rate_limited_until: datetime.datetime | Unset = UNSET
    webhook: LeadAdsWebhook | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.lead_ads_page import LeadAdsPage # noqa: PLC0415
        from ..models.lead_ads_receiving import LeadAdsReceiving # noqa: PLC0415
        from ..models.lead_ads_webhook import LeadAdsWebhook # noqa: PLC0415
        app_id = self.app_id

        can_manage = self.can_manage

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        helpers = self.helpers



        id = self.id

        leads30d = self.leads30d

        name = self.name

        pages = []
        for pages_item_data in self.pages:
            pages_item = pages_item_data.to_dict()
            pages.append(pages_item)



        platform = self.platform.value

        receiving = self.receiving.to_dict()

        status = self.status.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        disconnected_at: str | Unset = UNSET
        if not isinstance(self.disconnected_at, Unset):
            disconnected_at = self.disconnected_at.isoformat()

        problem: str | Unset = UNSET
        if not isinstance(self.problem, Unset):
            problem = self.problem.value


        rate_limited_until: str | Unset = UNSET
        if not isinstance(self.rate_limited_until, Unset):
            rate_limited_until = self.rate_limited_until.isoformat()

        webhook: dict[str, Any] | Unset = UNSET
        if not isinstance(self.webhook, Unset):
            webhook = self.webhook.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "appId": app_id,
            "canManage": can_manage,
            "createdAt": created_at,
            "createdBy": created_by,
            "helpers": helpers,
            "id": id,
            "leads30d": leads30d,
            "name": name,
            "pages": pages,
            "platform": platform,
            "receiving": receiving,
            "status": status,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if disconnected_at is not UNSET:
            field_dict["disconnectedAt"] = disconnected_at
        if problem is not UNSET:
            field_dict["problem"] = problem
        if rate_limited_until is not UNSET:
            field_dict["rateLimitedUntil"] = rate_limited_until
        if webhook is not UNSET:
            field_dict["webhook"] = webhook

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.lead_ads_page import LeadAdsPage # noqa: PLC0415
        from ..models.lead_ads_receiving import LeadAdsReceiving # noqa: PLC0415
        from ..models.lead_ads_webhook import LeadAdsWebhook # noqa: PLC0415
        d = dict(src_dict)
        app_id = d.pop("appId")

        can_manage = d.pop("canManage")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        helpers = cast(list[str], d.pop("helpers"))


        id = d.pop("id")

        leads30d = d.pop("leads30d")

        name = d.pop("name")

        pages = []
        _pages = d.pop("pages")
        for pages_item_data in (_pages):
            pages_item = LeadAdsPage.from_dict(pages_item_data)



            pages.append(pages_item)


        platform = LeadAdsConnectionPlatform(d.pop("platform"))




        receiving = LeadAdsReceiving.from_dict(d.pop("receiving"))




        status = LeadAdsConnectionStatus(d.pop("status"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        _disconnected_at = d.pop("disconnectedAt", UNSET)
        disconnected_at: datetime.datetime | Unset
        if isinstance(_disconnected_at,  Unset):
            disconnected_at = UNSET
        else:
            disconnected_at = datetime.datetime.fromisoformat(_disconnected_at)




        _problem = d.pop("problem", UNSET)
        problem: LeadAdsConnectionProblem | Unset
        if isinstance(_problem,  Unset):
            problem = UNSET
        else:
            problem = LeadAdsConnectionProblem(_problem)




        _rate_limited_until = d.pop("rateLimitedUntil", UNSET)
        rate_limited_until: datetime.datetime | Unset
        if isinstance(_rate_limited_until,  Unset):
            rate_limited_until = UNSET
        else:
            rate_limited_until = datetime.datetime.fromisoformat(_rate_limited_until)




        _webhook = d.pop("webhook", UNSET)
        webhook: LeadAdsWebhook | Unset
        if isinstance(_webhook,  Unset):
            webhook = UNSET
        else:
            webhook = LeadAdsWebhook.from_dict(_webhook)




        lead_ads_connection = cls(
            app_id=app_id,
            can_manage=can_manage,
            created_at=created_at,
            created_by=created_by,
            helpers=helpers,
            id=id,
            leads30d=leads30d,
            name=name,
            pages=pages,
            platform=platform,
            receiving=receiving,
            status=status,
            checked_at=checked_at,
            disconnected_at=disconnected_at,
            problem=problem,
            rate_limited_until=rate_limited_until,
            webhook=webhook,
        )

        return lead_ads_connection

