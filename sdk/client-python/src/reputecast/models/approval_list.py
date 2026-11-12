from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.approval_counts import ApprovalCounts
  from ..models.approval_filter_option import ApprovalFilterOption
  from ..models.approval_list_item import ApprovalListItem
  from ..models.approval_policy_summary import ApprovalPolicySummary





T = TypeVar("T", bound="ApprovalList")



@_attrs_define
class ApprovalList:
    """ 
        Attributes:
            all_requests (bool):
            approvers (list[ApprovalFilterOption]):
            authors (list[ApprovalFilterOption]):
            can_manage (bool):
            counts (ApprovalCounts):
            items (list[ApprovalListItem]):
            policy (ApprovalPolicySummary):
            next_cursor (str | Unset):
     """

    all_requests: bool
    approvers: list[ApprovalFilterOption]
    authors: list[ApprovalFilterOption]
    can_manage: bool
    counts: ApprovalCounts
    items: list[ApprovalListItem]
    policy: ApprovalPolicySummary
    next_cursor: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_counts import ApprovalCounts # noqa: PLC0415
        from ..models.approval_filter_option import ApprovalFilterOption # noqa: PLC0415
        from ..models.approval_list_item import ApprovalListItem # noqa: PLC0415
        from ..models.approval_policy_summary import ApprovalPolicySummary # noqa: PLC0415
        all_requests = self.all_requests

        approvers = []
        for approvers_item_data in self.approvers:
            approvers_item = approvers_item_data.to_dict()
            approvers.append(approvers_item)



        authors = []
        for authors_item_data in self.authors:
            authors_item = authors_item_data.to_dict()
            authors.append(authors_item)



        can_manage = self.can_manage

        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        policy = self.policy.to_dict()

        next_cursor = self.next_cursor


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allRequests": all_requests,
            "approvers": approvers,
            "authors": authors,
            "canManage": can_manage,
            "counts": counts,
            "items": items,
            "policy": policy,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_counts import ApprovalCounts # noqa: PLC0415
        from ..models.approval_filter_option import ApprovalFilterOption # noqa: PLC0415
        from ..models.approval_list_item import ApprovalListItem # noqa: PLC0415
        from ..models.approval_policy_summary import ApprovalPolicySummary # noqa: PLC0415
        d = dict(src_dict)
        all_requests = d.pop("allRequests")

        approvers = []
        _approvers = d.pop("approvers")
        for approvers_item_data in (_approvers):
            approvers_item = ApprovalFilterOption.from_dict(approvers_item_data)



            approvers.append(approvers_item)


        authors = []
        _authors = d.pop("authors")
        for authors_item_data in (_authors):
            authors_item = ApprovalFilterOption.from_dict(authors_item_data)



            authors.append(authors_item)


        can_manage = d.pop("canManage")

        counts = ApprovalCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ApprovalListItem.from_dict(items_item_data)



            items.append(items_item)


        policy = ApprovalPolicySummary.from_dict(d.pop("policy"))




        next_cursor = d.pop("nextCursor", UNSET)

        approval_list = cls(
            all_requests=all_requests,
            approvers=approvers,
            authors=authors,
            can_manage=can_manage,
            counts=counts,
            items=items,
            policy=policy,
            next_cursor=next_cursor,
        )

        return approval_list

