from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_tracking_check_mode import ProviderTrackingCheckMode
from ..models.provider_tracking_check_status import ProviderTrackingCheckStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderTrackingCheck")



@_attrs_define
class ProviderTrackingCheck:
    """ 
        Attributes:
            can_disable (bool):
            detail (str):
            link_rewrite (bool):
            mode (ProviderTrackingCheckMode):
            open_pixel (bool):
            provider (str):
            provider_id (str):
            scope (str):
            status (ProviderTrackingCheckStatus):
            checked_at (datetime.datetime | Unset):
     """

    can_disable: bool
    detail: str
    link_rewrite: bool
    mode: ProviderTrackingCheckMode
    open_pixel: bool
    provider: str
    provider_id: str
    scope: str
    status: ProviderTrackingCheckStatus
    checked_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        can_disable = self.can_disable

        detail = self.detail

        link_rewrite = self.link_rewrite

        mode = self.mode.value

        open_pixel = self.open_pixel

        provider = self.provider

        provider_id = self.provider_id

        scope = self.scope

        status = self.status.value

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canDisable": can_disable,
            "detail": detail,
            "linkRewrite": link_rewrite,
            "mode": mode,
            "openPixel": open_pixel,
            "provider": provider,
            "providerId": provider_id,
            "scope": scope,
            "status": status,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        can_disable = d.pop("canDisable")

        detail = d.pop("detail")

        link_rewrite = d.pop("linkRewrite")

        mode = ProviderTrackingCheckMode(d.pop("mode"))




        open_pixel = d.pop("openPixel")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        scope = d.pop("scope")

        status = ProviderTrackingCheckStatus(d.pop("status"))




        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        provider_tracking_check = cls(
            can_disable=can_disable,
            detail=detail,
            link_rewrite=link_rewrite,
            mode=mode,
            open_pixel=open_pixel,
            provider=provider,
            provider_id=provider_id,
            scope=scope,
            status=status,
            checked_at=checked_at,
        )

        return provider_tracking_check

