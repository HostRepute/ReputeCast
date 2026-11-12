from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.overview_refresh_failure import OverviewRefreshFailure
  from ..models.provider_card import ProviderCard





T = TypeVar("T", bound="OverviewProviderCard")



@_attrs_define
class OverviewProviderCard:
    """ 
        Attributes:
            card (ProviderCard):
            detail (str):
            display_name (str):
            refresh (OverviewRefreshFailure | Unset):
            status_label (str | Unset):
            status_url (str | Unset):
     """

    card: ProviderCard
    detail: str
    display_name: str
    refresh: OverviewRefreshFailure | Unset = UNSET
    status_label: str | Unset = UNSET
    status_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.overview_refresh_failure import OverviewRefreshFailure # noqa: PLC0415
        from ..models.provider_card import ProviderCard # noqa: PLC0415
        card = self.card.to_dict()

        detail = self.detail

        display_name = self.display_name

        refresh: dict[str, Any] | Unset = UNSET
        if not isinstance(self.refresh, Unset):
            refresh = self.refresh.to_dict()

        status_label = self.status_label

        status_url = self.status_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "card": card,
            "detail": detail,
            "displayName": display_name,
        })
        if refresh is not UNSET:
            field_dict["refresh"] = refresh
        if status_label is not UNSET:
            field_dict["statusLabel"] = status_label
        if status_url is not UNSET:
            field_dict["statusUrl"] = status_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.overview_refresh_failure import OverviewRefreshFailure # noqa: PLC0415
        from ..models.provider_card import ProviderCard # noqa: PLC0415
        d = dict(src_dict)
        card = ProviderCard.from_dict(d.pop("card"))




        detail = d.pop("detail")

        display_name = d.pop("displayName")

        _refresh = d.pop("refresh", UNSET)
        refresh: OverviewRefreshFailure | Unset
        if isinstance(_refresh,  Unset):
            refresh = UNSET
        else:
            refresh = OverviewRefreshFailure.from_dict(_refresh)




        status_label = d.pop("statusLabel", UNSET)

        status_url = d.pop("statusUrl", UNSET)

        overview_provider_card = cls(
            card=card,
            detail=detail,
            display_name=display_name,
            refresh=refresh,
            status_label=status_label,
            status_url=status_url,
        )

        return overview_provider_card

