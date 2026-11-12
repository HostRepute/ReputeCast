from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessRunFailure")



@_attrs_define
class ReadinessRunFailure:
    """ 
        Attributes:
            at (datetime.datetime):
            error (str):
            run_id (str):
     """

    at: datetime.datetime
    error: str
    run_id: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        error = self.error

        run_id = self.run_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "error": error,
            "runId": run_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        error = d.pop("error")

        run_id = d.pop("runId")

        readiness_run_failure = cls(
            at=at,
            error=error,
            run_id=run_id,
        )

        return readiness_run_failure

