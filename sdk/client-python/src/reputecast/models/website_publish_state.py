from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_publish_state_rule import WebsitePublishStateRule
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.website_post_approval import WebsitePostApproval





T = TypeVar("T", bound="WebsitePublishState")



@_attrs_define
class WebsitePublishState:
    """ 
        Attributes:
            approvers (list[str]):
            blocked (bool):
            can_publish (bool):
            needs_approval (bool):
            approved_id (str | Unset):
            block_reason (str | Unset):
            pending (WebsitePostApproval | Unset):
            rule (WebsitePublishStateRule | Unset):
     """

    approvers: list[str]
    blocked: bool
    can_publish: bool
    needs_approval: bool
    approved_id: str | Unset = UNSET
    block_reason: str | Unset = UNSET
    pending: WebsitePostApproval | Unset = UNSET
    rule: WebsitePublishStateRule | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.website_post_approval import WebsitePostApproval # noqa: PLC0415
        approvers = self.approvers



        blocked = self.blocked

        can_publish = self.can_publish

        needs_approval = self.needs_approval

        approved_id = self.approved_id

        block_reason = self.block_reason

        pending: dict[str, Any] | Unset = UNSET
        if not isinstance(self.pending, Unset):
            pending = self.pending.to_dict()

        rule: str | Unset = UNSET
        if not isinstance(self.rule, Unset):
            rule = self.rule.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "approvers": approvers,
            "blocked": blocked,
            "canPublish": can_publish,
            "needsApproval": needs_approval,
        })
        if approved_id is not UNSET:
            field_dict["approvedId"] = approved_id
        if block_reason is not UNSET:
            field_dict["blockReason"] = block_reason
        if pending is not UNSET:
            field_dict["pending"] = pending
        if rule is not UNSET:
            field_dict["rule"] = rule

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.website_post_approval import WebsitePostApproval # noqa: PLC0415
        d = dict(src_dict)
        approvers = cast(list[str], d.pop("approvers"))


        blocked = d.pop("blocked")

        can_publish = d.pop("canPublish")

        needs_approval = d.pop("needsApproval")

        approved_id = d.pop("approvedId", UNSET)

        block_reason = d.pop("blockReason", UNSET)

        _pending = d.pop("pending", UNSET)
        pending: WebsitePostApproval | Unset
        if isinstance(_pending,  Unset):
            pending = UNSET
        else:
            pending = WebsitePostApproval.from_dict(_pending)




        _rule = d.pop("rule", UNSET)
        rule: WebsitePublishStateRule | Unset
        if isinstance(_rule,  Unset):
            rule = UNSET
        else:
            rule = WebsitePublishStateRule(_rule)




        website_publish_state = cls(
            approvers=approvers,
            blocked=blocked,
            can_publish=can_publish,
            needs_approval=needs_approval,
            approved_id=approved_id,
            block_reason=block_reason,
            pending=pending,
            rule=rule,
        )

        return website_publish_state

