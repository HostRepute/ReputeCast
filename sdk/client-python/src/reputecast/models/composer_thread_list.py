from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.composer_thread import ComposerThread





T = TypeVar("T", bound="ComposerThreadList")



@_attrs_define
class ComposerThreadList:
    """ 
        Attributes:
            open_ (int):
            resolved (int):
            threads (list[ComposerThread]):
     """

    open_: int
    resolved: int
    threads: list[ComposerThread]





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_thread import ComposerThread # noqa: PLC0415
        open_ = self.open_

        resolved = self.resolved

        threads = []
        for threads_item_data in self.threads:
            threads_item = threads_item_data.to_dict()
            threads.append(threads_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "open": open_,
            "resolved": resolved,
            "threads": threads,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_thread import ComposerThread # noqa: PLC0415
        d = dict(src_dict)
        open_ = d.pop("open")

        resolved = d.pop("resolved")

        threads = []
        _threads = d.pop("threads")
        for threads_item_data in (_threads):
            threads_item = ComposerThread.from_dict(threads_item_data)



            threads.append(threads_item)


        composer_thread_list = cls(
            open_=open_,
            resolved=resolved,
            threads=threads,
        )

        return composer_thread_list

