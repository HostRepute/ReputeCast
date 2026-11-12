from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="RoleImpactPending")



@_attrs_define
class RoleImpactPending:
    """ 
        Attributes:
            requester (str):
            still_matches (list[str]):
            subject (str):
     """

    requester: str
    still_matches: list[str]
    subject: str





    def to_dict(self) -> dict[str, Any]:
        requester = self.requester

        still_matches = self.still_matches



        subject = self.subject


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "requester": requester,
            "stillMatches": still_matches,
            "subject": subject,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        requester = d.pop("requester")

        still_matches = cast(list[str], d.pop("stillMatches"))


        subject = d.pop("subject")

        role_impact_pending = cls(
            requester=requester,
            still_matches=still_matches,
            subject=subject,
        )

        return role_impact_pending

