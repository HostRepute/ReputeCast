from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.product_file import ProductFile





T = TypeVar("T", bound="FormDownload")



@_attrs_define
class FormDownload:
    """ 
        Attributes:
            days (int):
            downloaded (int):
            links (int):
            file (ProductFile | Unset):
     """

    days: int
    downloaded: int
    links: int
    file: ProductFile | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_file import ProductFile # noqa: PLC0415
        days = self.days

        downloaded = self.downloaded

        links = self.links

        file: dict[str, Any] | Unset = UNSET
        if not isinstance(self.file, Unset):
            file = self.file.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "downloaded": downloaded,
            "links": links,
        })
        if file is not UNSET:
            field_dict["file"] = file

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_file import ProductFile # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        downloaded = d.pop("downloaded")

        links = d.pop("links")

        _file = d.pop("file", UNSET)
        file: ProductFile | Unset
        if isinstance(_file,  Unset):
            file = UNSET
        else:
            file = ProductFile.from_dict(_file)




        form_download = cls(
            days=days,
            downloaded=downloaded,
            links=links,
            file=file,
        )

        return form_download

