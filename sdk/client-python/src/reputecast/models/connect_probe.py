from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConnectProbe")



@_attrs_define
class ConnectProbe:
    """ 
        Attributes:
            at (datetime.datetime):
            ok (bool):
            summary (str):
            transcript (str):
            error (str | Unset):
     """

    at: datetime.datetime
    ok: bool
    summary: str
    transcript: str
    error: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        ok = self.ok

        summary = self.summary

        transcript = self.transcript

        error = self.error


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "ok": ok,
            "summary": summary,
            "transcript": transcript,
        })
        if error is not UNSET:
            field_dict["error"] = error

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        ok = d.pop("ok")

        summary = d.pop("summary")

        transcript = d.pop("transcript")

        error = d.pop("error", UNSET)

        connect_probe = cls(
            at=at,
            ok=ok,
            summary=summary,
            transcript=transcript,
            error=error,
        )

        return connect_probe

