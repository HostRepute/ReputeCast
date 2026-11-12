from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_overview_other import SmsOverviewOther
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_catalog_item import SmsCatalogItem
  from ..models.sms_overview_tiles import SmsOverviewTiles
  from ..models.sms_provider_card import SmsProviderCard
  from ..models.sms_route_summary import SmsRouteSummary
  from ..models.sms_sender_row import SmsSenderRow





T = TypeVar("T", bound="SmsOverview")



@_attrs_define
class SmsOverview:
    """ 
        Attributes:
            can_manage (bool):
            helpers (list[str]):
            not_connected (list[SmsCatalogItem]):
            other (SmsOverviewOther):
            providers (list[SmsProviderCard]):
            routes (list[SmsRouteSummary]):
            senders (list[SmsSenderRow]):
            tiles (SmsOverviewTiles):
            timezone (str):
            checked_at (datetime.datetime | Unset):
            other_sender (str | Unset):
     """

    can_manage: bool
    helpers: list[str]
    not_connected: list[SmsCatalogItem]
    other: SmsOverviewOther
    providers: list[SmsProviderCard]
    routes: list[SmsRouteSummary]
    senders: list[SmsSenderRow]
    tiles: SmsOverviewTiles
    timezone: str
    checked_at: datetime.datetime | Unset = UNSET
    other_sender: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_catalog_item import SmsCatalogItem # noqa: PLC0415
        from ..models.sms_overview_tiles import SmsOverviewTiles # noqa: PLC0415
        from ..models.sms_provider_card import SmsProviderCard # noqa: PLC0415
        from ..models.sms_route_summary import SmsRouteSummary # noqa: PLC0415
        from ..models.sms_sender_row import SmsSenderRow # noqa: PLC0415
        can_manage = self.can_manage

        helpers = self.helpers



        not_connected = []
        for not_connected_item_data in self.not_connected:
            not_connected_item = not_connected_item_data.to_dict()
            not_connected.append(not_connected_item)



        other = self.other.value

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        routes = []
        for routes_item_data in self.routes:
            routes_item = routes_item_data.to_dict()
            routes.append(routes_item)



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)



        tiles = self.tiles.to_dict()

        timezone = self.timezone

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        other_sender = self.other_sender


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "helpers": helpers,
            "notConnected": not_connected,
            "other": other,
            "providers": providers,
            "routes": routes,
            "senders": senders,
            "tiles": tiles,
            "timezone": timezone,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if other_sender is not UNSET:
            field_dict["otherSender"] = other_sender

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_catalog_item import SmsCatalogItem # noqa: PLC0415
        from ..models.sms_overview_tiles import SmsOverviewTiles # noqa: PLC0415
        from ..models.sms_provider_card import SmsProviderCard # noqa: PLC0415
        from ..models.sms_route_summary import SmsRouteSummary # noqa: PLC0415
        from ..models.sms_sender_row import SmsSenderRow # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        helpers = cast(list[str], d.pop("helpers"))


        not_connected = []
        _not_connected = d.pop("notConnected")
        for not_connected_item_data in (_not_connected):
            not_connected_item = SmsCatalogItem.from_dict(not_connected_item_data)



            not_connected.append(not_connected_item)


        other = SmsOverviewOther(d.pop("other"))




        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SmsProviderCard.from_dict(providers_item_data)



            providers.append(providers_item)


        routes = []
        _routes = d.pop("routes")
        for routes_item_data in (_routes):
            routes_item = SmsRouteSummary.from_dict(routes_item_data)



            routes.append(routes_item)


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SmsSenderRow.from_dict(senders_item_data)



            senders.append(senders_item)


        tiles = SmsOverviewTiles.from_dict(d.pop("tiles"))




        timezone = d.pop("timezone")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        other_sender = d.pop("otherSender", UNSET)

        sms_overview = cls(
            can_manage=can_manage,
            helpers=helpers,
            not_connected=not_connected,
            other=other,
            providers=providers,
            routes=routes,
            senders=senders,
            tiles=tiles,
            timezone=timezone,
            checked_at=checked_at,
            other_sender=other_sender,
        )

        return sms_overview

