from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.transactional_batch_item import TransactionalBatchItem





T = TypeVar("T", bound="TransactionalBatchResult")



@_attrs_define
class TransactionalBatchResult:
    """ 
        Attributes:
            accepted (int):
            failed (int):
            results (list[TransactionalBatchItem]):
     """

    accepted: int
    failed: int
    results: list[TransactionalBatchItem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.transactional_batch_item import TransactionalBatchItem # noqa: PLC0415
        accepted = self.accepted

        failed = self.failed

        results = []
        for results_item_data in self.results:
            results_item = results_item_data.to_dict()
            results.append(results_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "failed": failed,
            "results": results,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.transactional_batch_item import TransactionalBatchItem # noqa: PLC0415
        d = dict(src_dict)
        accepted = d.pop("accepted")

        failed = d.pop("failed")

        results = []
        _results = d.pop("results")
        for results_item_data in (_results):
            results_item = TransactionalBatchItem.from_dict(results_item_data)



            results.append(results_item)


        transactional_batch_result = cls(
            accepted=accepted,
            failed=failed,
            results=results,
        )

        return transactional_batch_result

