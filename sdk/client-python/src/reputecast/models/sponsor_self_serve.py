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
  from ..models.sponsor_kit_stats import SponsorKitStats
  from ..models.sponsor_media_kit import SponsorMediaKit





T = TypeVar("T", bound="SponsorSelfServe")



@_attrs_define
class SponsorSelfServe:
    """ 
        Attributes:
            can_manage (bool):
            contact_email (str):
            helpers (list[str]):
            media_kit (SponsorMediaKit):
            page_url (str):
            payments_connected (bool):
            payments_test (bool):
            self_serve (bool):
            slots (int):
            stats (list[SponsorKitStats]):
            updated_at (datetime.datetime | Unset):
     """

    can_manage: bool
    contact_email: str
    helpers: list[str]
    media_kit: SponsorMediaKit
    page_url: str
    payments_connected: bool
    payments_test: bool
    self_serve: bool
    slots: int
    stats: list[SponsorKitStats]
    updated_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_kit_stats import SponsorKitStats # noqa: PLC0415
        from ..models.sponsor_media_kit import SponsorMediaKit # noqa: PLC0415
        can_manage = self.can_manage

        contact_email = self.contact_email

        helpers = self.helpers



        media_kit = self.media_kit.to_dict()

        page_url = self.page_url

        payments_connected = self.payments_connected

        payments_test = self.payments_test

        self_serve = self.self_serve

        slots = self.slots

        stats = []
        for stats_item_data in self.stats:
            stats_item = stats_item_data.to_dict()
            stats.append(stats_item)



        updated_at: str | Unset = UNSET
        if not isinstance(self.updated_at, Unset):
            updated_at = self.updated_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "contactEmail": contact_email,
            "helpers": helpers,
            "mediaKit": media_kit,
            "pageUrl": page_url,
            "paymentsConnected": payments_connected,
            "paymentsTest": payments_test,
            "selfServe": self_serve,
            "slots": slots,
            "stats": stats,
        })
        if updated_at is not UNSET:
            field_dict["updatedAt"] = updated_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_kit_stats import SponsorKitStats # noqa: PLC0415
        from ..models.sponsor_media_kit import SponsorMediaKit # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        contact_email = d.pop("contactEmail")

        helpers = cast(list[str], d.pop("helpers"))


        media_kit = SponsorMediaKit.from_dict(d.pop("mediaKit"))




        page_url = d.pop("pageUrl")

        payments_connected = d.pop("paymentsConnected")

        payments_test = d.pop("paymentsTest")

        self_serve = d.pop("selfServe")

        slots = d.pop("slots")

        stats = []
        _stats = d.pop("stats")
        for stats_item_data in (_stats):
            stats_item = SponsorKitStats.from_dict(stats_item_data)



            stats.append(stats_item)


        _updated_at = d.pop("updatedAt", UNSET)
        updated_at: datetime.datetime | Unset
        if isinstance(_updated_at,  Unset):
            updated_at = UNSET
        else:
            updated_at = datetime.datetime.fromisoformat(_updated_at)




        sponsor_self_serve = cls(
            can_manage=can_manage,
            contact_email=contact_email,
            helpers=helpers,
            media_kit=media_kit,
            page_url=page_url,
            payments_connected=payments_connected,
            payments_test=payments_test,
            self_serve=self_serve,
            slots=slots,
            stats=stats,
            updated_at=updated_at,
        )

        return sponsor_self_serve

