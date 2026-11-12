from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ApprovalPolicyMatches")



@_attrs_define
class ApprovalPolicyMatches:
    """ 
        Attributes:
            approval_lists (list[str]):
            lacks_send (list[str]):
            lacks_send_roles (list[str]):
            warming (str):
     """

    approval_lists: list[str]
    lacks_send: list[str]
    lacks_send_roles: list[str]
    warming: str





    def to_dict(self) -> dict[str, Any]:
        approval_lists = self.approval_lists



        lacks_send = self.lacks_send



        lacks_send_roles = self.lacks_send_roles



        warming = self.warming


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvalLists": approval_lists,
            "lacksSend": lacks_send,
            "lacksSendRoles": lacks_send_roles,
            "warming": warming,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approval_lists = cast(list[str], d.pop("approvalLists"))


        lacks_send = cast(list[str], d.pop("lacksSend"))


        lacks_send_roles = cast(list[str], d.pop("lacksSendRoles"))


        warming = d.pop("warming")

        approval_policy_matches = cls(
            approval_lists=approval_lists,
            lacks_send=lacks_send,
            lacks_send_roles=lacks_send_roles,
            warming=warming,
        )

        return approval_policy_matches

