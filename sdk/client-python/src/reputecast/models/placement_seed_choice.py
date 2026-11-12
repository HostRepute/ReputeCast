from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PlacementSeedChoice")



@_attrs_define
class PlacementSeedChoice:
    """ 
        Attributes:
            email (str):
            id (str):
            provider (str):
            provider_name (str):
            selected (bool):
            status (str):
            status_at (datetime.datetime | Unset):
     """

    email: str
    id: str
    provider: str
    provider_name: str
    selected: bool
    status: str
    status_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        id = self.id

        provider = self.provider

        provider_name = self.provider_name

        selected = self.selected

        status = self.status

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "id": id,
            "provider": provider,
            "providerName": provider_name,
            "selected": selected,
            "status": status,
        })
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        id = d.pop("id")

        provider = d.pop("provider")

        provider_name = d.pop("providerName")

        selected = d.pop("selected")

        status = d.pop("status")

        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        placement_seed_choice = cls(
            email=email,
            id=id,
            provider=provider,
            provider_name=provider_name,
            selected=selected,
            status=status,
            status_at=status_at,
        )

        return placement_seed_choice

