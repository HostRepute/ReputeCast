from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.approval_share import ApprovalShare





T = TypeVar("T", bound="ApprovalRouteView")



@_attrs_define
class ApprovalRouteView:
    """ 
        Attributes:
            shares (list[ApprovalShare]):
            split_label (str):
     """

    shares: list[ApprovalShare]
    split_label: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_share import ApprovalShare # noqa: PLC0415
        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)



        split_label = self.split_label


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "shares": shares,
            "splitLabel": split_label,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_share import ApprovalShare # noqa: PLC0415
        d = dict(src_dict)
        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = ApprovalShare.from_dict(shares_item_data)



            shares.append(shares_item)


        split_label = d.pop("splitLabel")

        approval_route_view = cls(
            shares=shares,
            split_label=split_label,
        )

        return approval_route_view

