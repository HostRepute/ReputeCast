from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="UpdateDomainInput")



@_attrs_define
class UpdateDomainInput:
    """ 
        Attributes:
            note (str | Unset):
            tracking_domain (str | Unset):
            transactional (bool | Unset):
     """

    note: str | Unset = UNSET
    tracking_domain: str | Unset = UNSET
    transactional: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        note = self.note

        tracking_domain = self.tracking_domain

        transactional = self.transactional


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if note is not UNSET:
            field_dict["note"] = note
        if tracking_domain is not UNSET:
            field_dict["trackingDomain"] = tracking_domain
        if transactional is not UNSET:
            field_dict["transactional"] = transactional

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        note = d.pop("note", UNSET)

        tracking_domain = d.pop("trackingDomain", UNSET)

        transactional = d.pop("transactional", UNSET)

        update_domain_input = cls(
            note=note,
            tracking_domain=tracking_domain,
            transactional=transactional,
        )

        return update_domain_input

