from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.word_press_check_outcome import WordPressCheckOutcome
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WordPressCheck")



@_attrs_define
class WordPressCheck:
    """ 
        Attributes:
            checked_at (datetime.datetime):
            detail (str):  Example: Timed out after 10 s.
            ok (bool):
            outcome (WordPressCheckOutcome):
            ms (int | Unset):
            seconds (float | Unset):
            status (int | Unset):
     """

    checked_at: datetime.datetime
    detail: str
    ok: bool
    outcome: WordPressCheckOutcome
    ms: int | Unset = UNSET
    seconds: float | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checked_at = self.checked_at.isoformat()

        detail = self.detail

        ok = self.ok

        outcome = self.outcome.value

        ms = self.ms

        seconds = self.seconds

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checkedAt": checked_at,
            "detail": detail,
            "ok": ok,
            "outcome": outcome,
        })
        if ms is not UNSET:
            field_dict["ms"] = ms
        if seconds is not UNSET:
            field_dict["seconds"] = seconds
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked_at = datetime.datetime.fromisoformat(d.pop("checkedAt"))




        detail = d.pop("detail")

        ok = d.pop("ok")

        outcome = WordPressCheckOutcome(d.pop("outcome"))




        ms = d.pop("ms", UNSET)

        seconds = d.pop("seconds", UNSET)

        status = d.pop("status", UNSET)

        word_press_check = cls(
            checked_at=checked_at,
            detail=detail,
            ok=ok,
            outcome=outcome,
            ms=ms,
            seconds=seconds,
            status=status,
        )

        return word_press_check

