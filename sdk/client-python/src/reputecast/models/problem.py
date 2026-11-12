from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.error_detail import ErrorDetail
  from ..models.problem_meta import ProblemMeta





T = TypeVar("T", bound="Problem")



@_attrs_define
class Problem:
    """ 
        Attributes:
            code (str | Unset): Stable error code, e.g. permission_denied
            detail (str | Unset): A human-readable explanation specific to this occurrence of the problem. Example: Property
                foo is required but is missing..
            errors (list[ErrorDetail] | Unset): Optional list of individual error details
            helpers (list[str] | Unset): Members who hold the missing permission
            instance (str | Unset): A URI reference that identifies the specific occurrence of the problem. Example:
                https://example.com/error-log/abc123.
            meta (ProblemMeta | Unset):
            permission (str | Unset): The missing permission key
            status (int | Unset): HTTP status code Example: 400.
            title (str | Unset): A short, human-readable summary of the problem type. This value should not change between
                occurrences of the error. Example: Bad Request.
            type_ (str | Unset): A URI reference to human-readable documentation for the error. Default: 'about:blank'.
                Example: https://example.com/errors/example.
     """

    code: str | Unset = UNSET
    detail: str | Unset = UNSET
    errors: list[ErrorDetail] | Unset = UNSET
    helpers: list[str] | Unset = UNSET
    instance: str | Unset = UNSET
    meta: ProblemMeta | Unset = UNSET
    permission: str | Unset = UNSET
    status: int | Unset = UNSET
    title: str | Unset = UNSET
    type_: str | Unset = 'about:blank'





    def to_dict(self) -> dict[str, Any]:
        from ..models.error_detail import ErrorDetail # noqa: PLC0415
        from ..models.problem_meta import ProblemMeta # noqa: PLC0415
        code = self.code

        detail = self.detail

        errors: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.errors, Unset):
            errors = []
            for errors_item_data in self.errors:
                errors_item = errors_item_data.to_dict()
                errors.append(errors_item)



        helpers: list[str] | Unset = UNSET
        if not isinstance(self.helpers, Unset):
            helpers = self.helpers



        instance = self.instance

        meta: dict[str, Any] | Unset = UNSET
        if not isinstance(self.meta, Unset):
            meta = self.meta.to_dict()

        permission = self.permission

        status = self.status

        title = self.title

        type_ = self.type_


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if code is not UNSET:
            field_dict["code"] = code
        if detail is not UNSET:
            field_dict["detail"] = detail
        if errors is not UNSET:
            field_dict["errors"] = errors
        if helpers is not UNSET:
            field_dict["helpers"] = helpers
        if instance is not UNSET:
            field_dict["instance"] = instance
        if meta is not UNSET:
            field_dict["meta"] = meta
        if permission is not UNSET:
            field_dict["permission"] = permission
        if status is not UNSET:
            field_dict["status"] = status
        if title is not UNSET:
            field_dict["title"] = title
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.error_detail import ErrorDetail # noqa: PLC0415
        from ..models.problem_meta import ProblemMeta # noqa: PLC0415
        d = dict(src_dict)
        code = d.pop("code", UNSET)

        detail = d.pop("detail", UNSET)

        _errors = d.pop("errors", UNSET)
        errors: list[ErrorDetail] | Unset = UNSET
        if _errors is not UNSET:
            errors = []
            for errors_item_data in _errors:
                errors_item = ErrorDetail.from_dict(errors_item_data)



                errors.append(errors_item)


        helpers = cast(list[str], d.pop("helpers", UNSET))


        instance = d.pop("instance", UNSET)

        _meta = d.pop("meta", UNSET)
        meta: ProblemMeta | Unset
        if isinstance(_meta,  Unset):
            meta = UNSET
        else:
            meta = ProblemMeta.from_dict(_meta)




        permission = d.pop("permission", UNSET)

        status = d.pop("status", UNSET)

        title = d.pop("title", UNSET)

        type_ = d.pop("type", UNSET)

        problem = cls(
            code=code,
            detail=detail,
            errors=errors,
            helpers=helpers,
            instance=instance,
            meta=meta,
            permission=permission,
            status=status,
            title=title,
            type_=type_,
        )

        return problem

