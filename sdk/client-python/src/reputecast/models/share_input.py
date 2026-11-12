from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ShareInput")



@_attrs_define
class ShareInput:
    """ 
        Attributes:
            expires_in_days (int | Unset): 30 when absent
            include_cost (bool | Unset): Show the cost per sending provider
            include_email (bool | Unset): Show the email as sent, in the click map
            include_warmup (bool | Unset): Show the warm-up day of each sending provider
     """

    expires_in_days: int | Unset = UNSET
    include_cost: bool | Unset = UNSET
    include_email: bool | Unset = UNSET
    include_warmup: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        expires_in_days = self.expires_in_days

        include_cost = self.include_cost

        include_email = self.include_email

        include_warmup = self.include_warmup


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if expires_in_days is not UNSET:
            field_dict["expiresInDays"] = expires_in_days
        if include_cost is not UNSET:
            field_dict["includeCost"] = include_cost
        if include_email is not UNSET:
            field_dict["includeEmail"] = include_email
        if include_warmup is not UNSET:
            field_dict["includeWarmup"] = include_warmup

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        expires_in_days = d.pop("expiresInDays", UNSET)

        include_cost = d.pop("includeCost", UNSET)

        include_email = d.pop("includeEmail", UNSET)

        include_warmup = d.pop("includeWarmup", UNSET)

        share_input = cls(
            expires_in_days=expires_in_days,
            include_cost=include_cost,
            include_email=include_email,
            include_warmup=include_warmup,
        )

        return share_input

