from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_bulk_failure import ContactBulkFailure





T = TypeVar("T", bound="ContactBulkResult")



@_attrs_define
class ContactBulkResult:
    """ 
        Attributes:
            action (str):
            done (int):
            failed (int):
            failures (list[ContactBulkFailure]): Up to 20 examples
            matched (int):
            job_id (str | Unset):
            note (str | Unset):
            target (str | Unset):
     """

    action: str
    done: int
    failed: int
    failures: list[ContactBulkFailure]
    matched: int
    job_id: str | Unset = UNSET
    note: str | Unset = UNSET
    target: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_bulk_failure import ContactBulkFailure # noqa: PLC0415
        action = self.action

        done = self.done

        failed = self.failed

        failures = []
        for failures_item_data in self.failures:
            failures_item = failures_item_data.to_dict()
            failures.append(failures_item)



        matched = self.matched

        job_id = self.job_id

        note = self.note

        target = self.target


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "done": done,
            "failed": failed,
            "failures": failures,
            "matched": matched,
        })
        if job_id is not UNSET:
            field_dict["jobId"] = job_id
        if note is not UNSET:
            field_dict["note"] = note
        if target is not UNSET:
            field_dict["target"] = target

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_bulk_failure import ContactBulkFailure # noqa: PLC0415
        d = dict(src_dict)
        action = d.pop("action")

        done = d.pop("done")

        failed = d.pop("failed")

        failures = []
        _failures = d.pop("failures")
        for failures_item_data in (_failures):
            failures_item = ContactBulkFailure.from_dict(failures_item_data)



            failures.append(failures_item)


        matched = d.pop("matched")

        job_id = d.pop("jobId", UNSET)

        note = d.pop("note", UNSET)

        target = d.pop("target", UNSET)

        contact_bulk_result = cls(
            action=action,
            done=done,
            failed=failed,
            failures=failures,
            matched=matched,
            job_id=job_id,
            note=note,
            target=target,
        )

        return contact_bulk_result

