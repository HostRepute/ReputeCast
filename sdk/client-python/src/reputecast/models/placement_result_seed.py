from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PlacementResultSeed")



@_attrs_define
class PlacementResultSeed:
    """ 
        Attributes:
            email (str):
            position (int):
            provider (str):
            provider_name (str):
            status (str): The seed's status now (removed when it's gone)
            id (str | Unset):
            status_at (datetime.datetime | Unset):
     """

    email: str
    position: int
    provider: str
    provider_name: str
    status: str
    id: str | Unset = UNSET
    status_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        position = self.position

        provider = self.provider

        provider_name = self.provider_name

        status = self.status

        id = self.id

        status_at: str | Unset = UNSET
        if not isinstance(self.status_at, Unset):
            status_at = self.status_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "position": position,
            "provider": provider,
            "providerName": provider_name,
            "status": status,
        })
        if id is not UNSET:
            field_dict["id"] = id
        if status_at is not UNSET:
            field_dict["statusAt"] = status_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        position = d.pop("position")

        provider = d.pop("provider")

        provider_name = d.pop("providerName")

        status = d.pop("status")

        id = d.pop("id", UNSET)

        _status_at = d.pop("statusAt", UNSET)
        status_at: datetime.datetime | Unset
        if isinstance(_status_at,  Unset):
            status_at = UNSET
        else:
            status_at = datetime.datetime.fromisoformat(_status_at)




        placement_result_seed = cls(
            email=email,
            position=position,
            provider=provider,
            provider_name=provider_name,
            status=status,
            id=id,
            status_at=status_at,
        )

        return placement_result_seed

