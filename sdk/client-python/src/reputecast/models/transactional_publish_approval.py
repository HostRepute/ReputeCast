from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="TransactionalPublishApproval")



@_attrs_define
class TransactionalPublishApproval:
    """ 
        Attributes:
            approvers (list[str]):
            required (bool):
            reason (str | Unset):
     """

    approvers: list[str]
    required: bool
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approvers = self.approvers



        required = self.required

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "required": required,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        required = d.pop("required")

        reason = d.pop("reason", UNSET)

        transactional_publish_approval = cls(
            approvers=approvers,
            required=required,
            reason=reason,
        )

        return transactional_publish_approval

