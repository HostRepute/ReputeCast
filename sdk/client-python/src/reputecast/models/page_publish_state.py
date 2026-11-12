from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.page_approval import PageApproval





T = TypeVar("T", bound="PagePublishState")



@_attrs_define
class PagePublishState:
    """ 
        Attributes:
            approvers (list[str]):
            can_edit (bool):
            can_manage (bool): Navigation, visibility and the paywall (website:manage)
            can_publish (bool):
            needs_approval (bool):
            publishers (list[str]):
            pending (PageApproval | Unset):
            reason (str | Unset):
            rule (str | Unset):
     """

    approvers: list[str]
    can_edit: bool
    can_manage: bool
    can_publish: bool
    needs_approval: bool
    publishers: list[str]
    pending: PageApproval | Unset = UNSET
    reason: str | Unset = UNSET
    rule: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_approval import PageApproval # noqa: PLC0415
        approvers = self.approvers



        can_edit = self.can_edit

        can_manage = self.can_manage

        can_publish = self.can_publish

        needs_approval = self.needs_approval

        publishers = self.publishers



        pending: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pending, Unset):
            pending = self.pending.to_dict()

        reason = self.reason

        rule = self.rule


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "canEdit": can_edit,
            "canManage": can_manage,
            "canPublish": can_publish,
            "needsApproval": needs_approval,
            "publishers": publishers,
        })
        if pending is not UNSET:
            field_dict["pending"] = pending
        if reason is not UNSET:
            field_dict["reason"] = reason
        if rule is not UNSET:
            field_dict["rule"] = rule

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_approval import PageApproval # noqa: PLC0415
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        can_edit = d.pop("canEdit")

        can_manage = d.pop("canManage")

        can_publish = d.pop("canPublish")

        needs_approval = d.pop("needsApproval")

        publishers = cast(list[str], d.pop("publishers"))


        _pending = d.pop("pending", UNSET)
        pending: PageApproval | Unset
        if isinstance(_pending,  Unset):
            pending = UNSET
        else:
            pending = PageApproval.from_dict(_pending)




        reason = d.pop("reason", UNSET)

        rule = d.pop("rule", UNSET)

        page_publish_state = cls(
            approvers=approvers,
            can_edit=can_edit,
            can_manage=can_manage,
            can_publish=can_publish,
            needs_approval=needs_approval,
            publishers=publishers,
            pending=pending,
            reason=reason,
            rule=rule,
        )

        return page_publish_state

