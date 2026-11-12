from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ApprovalClientSignoffState")



@_attrs_define
class ApprovalClientSignoffState:
    """ 
        Attributes:
            approvers (list[str]):
            met (bool):
            staff_approvals (list[str]):
            viewer_is_staff (bool):
            signed_off_by (str | Unset):
     """

    approvers: list[str]
    met: bool
    staff_approvals: list[str]
    viewer_is_staff: bool
    signed_off_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        approvers = self.approvers



        met = self.met

        staff_approvals = self.staff_approvals



        viewer_is_staff = self.viewer_is_staff

        signed_off_by = self.signed_off_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "met": met,
            "staffApprovals": staff_approvals,
            "viewerIsStaff": viewer_is_staff,
        })
        if signed_off_by is not UNSET:
            field_dict["signedOffBy"] = signed_off_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        met = d.pop("met")

        staff_approvals = cast(list[str], d.pop("staffApprovals"))


        viewer_is_staff = d.pop("viewerIsStaff")

        signed_off_by = d.pop("signedOffBy", UNSET)

        approval_client_signoff_state = cls(
            approvers=approvers,
            met=met,
            staff_approvals=staff_approvals,
            viewer_is_staff=viewer_is_staff,
            signed_off_by=signed_off_by,
        )

        return approval_client_signoff_state

