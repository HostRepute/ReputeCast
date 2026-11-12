from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.member_two_fa_state import MemberTwoFAState
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="MemberTwoFA")



@_attrs_define
class MemberTwoFA:
    """ 
        Attributes:
            state (MemberTwoFAState):
            grace_days (int | Unset):
            grace_until (datetime.datetime | Unset):
            method (str | Unset):
     """

    state: MemberTwoFAState
    grace_days: int | Unset = UNSET
    grace_until: datetime.datetime | Unset = UNSET
    method: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        state = self.state.value

        grace_days = self.grace_days

        grace_until: str | Unset = UNSET
        if not isinstance(self.grace_until, Unset):
            grace_until = self.grace_until.isoformat()

        method = self.method


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "state": state,
        })
        if grace_days is not UNSET:
            field_dict["graceDays"] = grace_days
        if grace_until is not UNSET:
            field_dict["graceUntil"] = grace_until
        if method is not UNSET:
            field_dict["method"] = method

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        state = MemberTwoFAState(d.pop("state"))




        grace_days = d.pop("graceDays", UNSET)

        _grace_until = d.pop("graceUntil", UNSET)
        grace_until: datetime.datetime | Unset
        if isinstance(_grace_until,  Unset):
            grace_until = UNSET
        else:
            grace_until = datetime.datetime.fromisoformat(_grace_until)




        method = d.pop("method", UNSET)

        member_two_fa = cls(
            state=state,
            grace_days=grace_days,
            grace_until=grace_until,
            method=method,
        )

        return member_two_fa

