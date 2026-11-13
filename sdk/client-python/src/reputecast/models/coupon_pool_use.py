from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_pool_use_kind import CouponPoolUseKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="CouponPoolUse")



@_attrs_define
class CouponPoolUse:
    """ 
        Attributes:
            block (bool):
            id (str):
            kind (CouponPoolUseKind):
            live (bool):
            name (str):
            recipients (int):
            email_id (str | Unset):
            email_name (str | Unset):
            lists (str | Unset):
            status (str | Unset):
     """

    block: bool
    id: str
    kind: CouponPoolUseKind
    live: bool
    name: str
    recipients: int
    email_id: str | Unset = UNSET
    email_name: str | Unset = UNSET
    lists: str | Unset = UNSET
    status: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        block = self.block

        id = self.id

        kind = self.kind.value

        live = self.live

        name = self.name

        recipients = self.recipients

        email_id = self.email_id

        email_name = self.email_name

        lists = self.lists

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "block": block,
            "id": id,
            "kind": kind,
            "live": live,
            "name": name,
            "recipients": recipients,
        })
        if email_id is not UNSET:
            field_dict["emailId"] = email_id
        if email_name is not UNSET:
            field_dict["emailName"] = email_name
        if lists is not UNSET:
            field_dict["lists"] = lists
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        block = d.pop("block")

        id = d.pop("id")

        kind = CouponPoolUseKind(d.pop("kind"))




        live = d.pop("live")

        name = d.pop("name")

        recipients = d.pop("recipients")

        email_id = d.pop("emailId", UNSET)

        email_name = d.pop("emailName", UNSET)

        lists = d.pop("lists", UNSET)

        status = d.pop("status", UNSET)

        coupon_pool_use = cls(
            block=block,
            id=id,
            kind=kind,
            live=live,
            name=name,
            recipients=recipients,
            email_id=email_id,
            email_name=email_name,
            lists=lists,
            status=status,
        )

        return coupon_pool_use

