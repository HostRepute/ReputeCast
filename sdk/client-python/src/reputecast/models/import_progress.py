from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ImportProgress")



@_attrs_define
class ImportProgress:
    """ 
        Attributes:
            batch (int):
            batch_size (int):
            batches (int):
            imported (int):
            merged (int):
            pending (int):
            processed (int):
            skipped (int):
            suppressed (int):
            total (int):
     """

    batch: int
    batch_size: int
    batches: int
    imported: int
    merged: int
    pending: int
    processed: int
    skipped: int
    suppressed: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        batch = self.batch

        batch_size = self.batch_size

        batches = self.batches

        imported = self.imported

        merged = self.merged

        pending = self.pending

        processed = self.processed

        skipped = self.skipped

        suppressed = self.suppressed

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "batch": batch,
            "batchSize": batch_size,
            "batches": batches,
            "imported": imported,
            "merged": merged,
            "pending": pending,
            "processed": processed,
            "skipped": skipped,
            "suppressed": suppressed,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        batch = d.pop("batch")

        batch_size = d.pop("batchSize")

        batches = d.pop("batches")

        imported = d.pop("imported")

        merged = d.pop("merged")

        pending = d.pop("pending")

        processed = d.pop("processed")

        skipped = d.pop("skipped")

        suppressed = d.pop("suppressed")

        total = d.pop("total")

        import_progress = cls(
            batch=batch,
            batch_size=batch_size,
            batches=batches,
            imported=imported,
            merged=merged,
            pending=pending,
            processed=processed,
            skipped=skipped,
            suppressed=suppressed,
            total=total,
        )

        return import_progress

