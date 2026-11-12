from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.overview_provider_card import OverviewProviderCard





T = TypeVar("T", bound="OverviewProviders")



@_attrs_define
class OverviewProviders:
    """ 
        Attributes:
            connected (int):
            month_start (datetime.datetime):
            providers (list[OverviewProviderCard]):
     """

    connected: int
    month_start: datetime.datetime
    providers: list[OverviewProviderCard]





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_provider_card import OverviewProviderCard # noqa: PLC0415
        connected = self.connected

        month_start = self.month_start.isoformat()

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connected": connected,
            "monthStart": month_start,
            "providers": providers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_provider_card import OverviewProviderCard # noqa: PLC0415
        d = dict(src_dict)
        connected = d.pop("connected")

        month_start = datetime.datetime.fromisoformat(d.pop("monthStart"))




        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = OverviewProviderCard.from_dict(providers_item_data)



            providers.append(providers_item)


        overview_providers = cls(
            connected=connected,
            month_start=month_start,
            providers=providers,
        )

        return overview_providers

