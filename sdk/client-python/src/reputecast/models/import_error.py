from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.import_bad_row import ImportBadRow





T = TypeVar("T", bound="ImportError_")



@_attrs_define
class ImportError_:
    """ 
        Attributes:
            code (str): source_auth_failed, source_rate_limited, source_unavailable, file_columns, file_unreadable
            message (str):
            bad_rows (list[ImportBadRow] | Unset):
            header (int | Unset):
            retry_at (datetime.datetime | Unset):
            row (int | Unset):
            status (int | Unset): HTTP status from the source (401, 429)
     """

    code: str
    message: str
    bad_rows: list[ImportBadRow] | Unset = UNSET
    header: int | Unset = UNSET
    retry_at: datetime.datetime | Unset = UNSET
    row: int | Unset = UNSET
    status: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.import_bad_row import ImportBadRow # noqa: PLC0415
        code = self.code

        message = self.message

        bad_rows: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.bad_rows, Unset):
            bad_rows = []
            for bad_rows_item_data in self.bad_rows:
                bad_rows_item = bad_rows_item_data.to_dict()
                bad_rows.append(bad_rows_item)



        header = self.header

        retry_at: str | Unset = UNSET
        if not isinstance(self.retry_at, Unset):
            retry_at = self.retry_at.isoformat()

        row = self.row

        status = self.status


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "message": message,
        })
        if bad_rows is not UNSET:
            field_dict["badRows"] = bad_rows
        if header is not UNSET:
            field_dict["header"] = header
        if retry_at is not UNSET:
            field_dict["retryAt"] = retry_at
        if row is not UNSET:
            field_dict["row"] = row
        if status is not UNSET:
            field_dict["status"] = status

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.import_bad_row import ImportBadRow # noqa: PLC0415
        d = dict(src_dict)
        code = d.pop("code")

        message = d.pop("message")

        _bad_rows = d.pop("badRows", UNSET)
        bad_rows: list[ImportBadRow] | Unset = UNSET
        if _bad_rows is not UNSET:
            bad_rows = []
            for bad_rows_item_data in _bad_rows:
                bad_rows_item = ImportBadRow.from_dict(bad_rows_item_data)



                bad_rows.append(bad_rows_item)


        header = d.pop("header", UNSET)

        _retry_at = d.pop("retryAt", UNSET)
        retry_at: datetime.datetime | Unset
        if isinstance(_retry_at,  Unset):
            retry_at = UNSET
        else:
            retry_at = datetime.datetime.fromisoformat(_retry_at)




        row = d.pop("row", UNSET)

        status = d.pop("status", UNSET)

        import_error = cls(
            code=code,
            message=message,
            bad_rows=bad_rows,
            header=header,
            retry_at=retry_at,
            row=row,
            status=status,
        )

        return import_error

