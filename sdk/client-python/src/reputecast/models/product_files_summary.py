from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductFilesSummary")



@_attrs_define
class ProductFilesSummary:
    """ 
        Attributes:
            bytes_ (int):
            count (int):
            kind (str): The first file's kind in capitals (PDF, ZIP, XLSX)
            pages (int): The first file's pages, for a PDF
     """

    bytes_: int
    count: int
    kind: str
    pages: int





    def to_dict(self) -> dict[str, Any]:
        bytes_ = self.bytes_

        count = self.count

        kind = self.kind

        pages = self.pages


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "count": count,
            "kind": kind,
            "pages": pages,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        count = d.pop("count")

        kind = d.pop("kind")

        pages = d.pop("pages")

        product_files_summary = cls(
            bytes_=bytes_,
            count=count,
            kind=kind,
            pages=pages,
        )

        return product_files_summary

