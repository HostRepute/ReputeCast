from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectDomainInput")



@_attrs_define
class ConnectDomainInput:
    """ 
        Attributes:
            domain (str | Unset):
            skip (bool | Unset):
     """

    domain: str | Unset = UNSET
    skip: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        skip = self.skip


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if domain is not UNSET:
            field_dict["domain"] = domain
        if skip is not UNSET:
            field_dict["skip"] = skip

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain", UNSET)

        skip = d.pop("skip", UNSET)

        connect_domain_input = cls(
            domain=domain,
            skip=skip,
        )

        return connect_domain_input

