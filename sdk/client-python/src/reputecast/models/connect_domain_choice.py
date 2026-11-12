from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ConnectDomainChoice")



@_attrs_define
class ConnectDomainChoice:
    """ 
        Attributes:
            domain_id (str):
            name (str):
            skipped (bool):
            suggested (str):
     """

    domain_id: str
    name: str
    skipped: bool
    suggested: str





    def to_dict(self) -> dict[str, Any]:
        domain_id = self.domain_id

        name = self.name

        skipped = self.skipped

        suggested = self.suggested


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domainId": domain_id,
            "name": name,
            "skipped": skipped,
            "suggested": suggested,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain_id = d.pop("domainId")

        name = d.pop("name")

        skipped = d.pop("skipped")

        suggested = d.pop("suggested")

        connect_domain_choice = cls(
            domain_id=domain_id,
            name=name,
            skipped=skipped,
            suggested=suggested,
        )

        return connect_domain_choice

