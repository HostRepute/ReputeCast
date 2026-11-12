from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_checklist_step_blocked import EarnChecklistStepBlocked
from ..models.earn_checklist_step_key import EarnChecklistStepKey
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="EarnChecklistStep")



@_attrs_define
class EarnChecklistStep:
    """ 
        Attributes:
            done (bool):
            key (EarnChecklistStepKey):
            account (str | Unset):
            at (datetime.datetime | Unset):
            blocked (EarnChecklistStepBlocked | Unset):
            count (int | Unset):
            detail (str | Unset):
            mode (str | Unset):
            name (str | Unset):
            price_line (str | Unset):
            total (int | Unset):
     """

    done: bool
    key: EarnChecklistStepKey
    account: str | Unset = UNSET
    at: datetime.datetime | Unset = UNSET
    blocked: EarnChecklistStepBlocked | Unset = UNSET
    count: int | Unset = UNSET
    detail: str | Unset = UNSET
    mode: str | Unset = UNSET
    name: str | Unset = UNSET
    price_line: str | Unset = UNSET
    total: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        done = self.done

        key = self.key.value

        account = self.account

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        blocked: str | Unset = UNSET
        if not isinstance(self.blocked, Unset):
            blocked = self.blocked.value


        count = self.count

        detail = self.detail

        mode = self.mode

        name = self.name

        price_line = self.price_line

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "done": done,
            "key": key,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if at is not UNSET:
            field_dict["at"] = at
        if blocked is not UNSET:
            field_dict["blocked"] = blocked
        if count is not UNSET:
            field_dict["count"] = count
        if detail is not UNSET:
            field_dict["detail"] = detail
        if mode is not UNSET:
            field_dict["mode"] = mode
        if name is not UNSET:
            field_dict["name"] = name
        if price_line is not UNSET:
            field_dict["priceLine"] = price_line
        if total is not UNSET:
            field_dict["total"] = total

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        done = d.pop("done")

        key = EarnChecklistStepKey(d.pop("key"))




        account = d.pop("account", UNSET)

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        _blocked = d.pop("blocked", UNSET)
        blocked: EarnChecklistStepBlocked | Unset
        if isinstance(_blocked,  Unset):
            blocked = UNSET
        else:
            blocked = EarnChecklistStepBlocked(_blocked)




        count = d.pop("count", UNSET)

        detail = d.pop("detail", UNSET)

        mode = d.pop("mode", UNSET)

        name = d.pop("name", UNSET)

        price_line = d.pop("priceLine", UNSET)

        total = d.pop("total", UNSET)

        earn_checklist_step = cls(
            done=done,
            key=key,
            account=account,
            at=at,
            blocked=blocked,
            count=count,
            detail=detail,
            mode=mode,
            name=name,
            price_line=price_line,
            total=total,
        )

        return earn_checklist_step

