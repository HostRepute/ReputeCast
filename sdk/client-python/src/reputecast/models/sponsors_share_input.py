from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SponsorsShareInput")



@_attrs_define
class SponsorsShareInput:
    """ 
        Attributes:
            expires_in_days (int | Unset): 0 keeps the link until it's revoked.
     """

    expires_in_days: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        expires_in_days = self.expires_in_days


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if expires_in_days is not UNSET:
            field_dict["expiresInDays"] = expires_in_days

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        expires_in_days = d.pop("expiresInDays", UNSET)

        sponsors_share_input = cls(
            expires_in_days=expires_in_days,
        )

        return sponsors_share_input

