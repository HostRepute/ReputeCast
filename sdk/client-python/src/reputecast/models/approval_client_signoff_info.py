from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ApprovalClientSignoffInfo")



@_attrs_define
class ApprovalClientSignoffInfo:
    """ 
        Attributes:
            agency (str):
            approvers (list[str]):
            staff (list[str]):
     """

    agency: str
    approvers: list[str]
    staff: list[str]





    def to_dict(self) -> dict[str, Any]:
        agency = self.agency

        approvers = self.approvers



        staff = self.staff




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "agency": agency,
            "approvers": approvers,
            "staff": staff,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        agency = d.pop("agency")

        approvers = cast(list[str], d.pop("approvers"))


        staff = cast(list[str], d.pop("staff"))


        approval_client_signoff_info = cls(
            agency=agency,
            approvers=approvers,
            staff=staff,
        )

        return approval_client_signoff_info

